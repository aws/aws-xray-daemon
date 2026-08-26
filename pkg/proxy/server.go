// Package proxy provides an http server to act as a signing proxy for SDKs calling AWS X-Ray APIs
package proxy

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-xray-daemon/pkg/cfg"
	"github.com/aws/aws-xray-daemon/pkg/conn"
	log "github.com/cihub/seelog"
)

const service = "xray"
const connHeader = "Connection"

// forbiddenBody is the response body returned for a request the proxy refuses to
// sign and forward.
const forbiddenBody = `{"message":"The X-Ray daemon proxy only forwards the GetSamplingRules and GetSamplingTargets operations."}`

// allowedOperations are the request paths of the X-Ray sampling APIs, the only
// operations the signing proxy forwards.
//
// The TCP listener signs every request it forwards with the daemon's own
// credentials and has no way to identify or authenticate its callers, so any
// operation it accepts is an operation that any caller able to reach the
// listener may perform with the daemon's IAM role. The listener exists so that a
// co-located SDK can reach the sampling APIs without holding credentials of its
// own, so it is limited to exactly those two operations. Everything else, in
// particular the trace read and control plane APIs, is rejected without being
// signed or forwarded.
var allowedOperations = map[string]struct{}{
	"/GetSamplingRules": {}, // GetSamplingRules
	"/SamplingTargets":  {}, // GetSamplingTargets
}

// Server represents HTTP server.
type Server struct {
	*http.Server
}

// operationFilter rejects requests for operations outside allowedOperations
// before they reach the signing proxy.
type operationFilter struct {
	proxy http.Handler
}

func (f *operationFilter) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	operation, allowed := allowedOperation(req)
	if !allowed {
		log.Warnf("Rejecting request on HTTP Proxy server: %v %v is not an X-Ray sampling operation",
			req.Method, requestPath(req))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		if _, err := w.Write([]byte(forbiddenBody)); err != nil {
			log.Debugf("Unable to write response on HTTP Proxy server: %v", err)
		}
		return
	}

	// Forward the canonical path so that an escaped or traversing path cannot
	// resolve to a different operation once the request has been signed.
	req.URL.Path = operation
	req.URL.RawPath = ""

	f.proxy.ServeHTTP(w, req)
}

// allowedOperation returns the canonical path of the sampling operation the
// request targets, and whether the request may be forwarded at all.
func allowedOperation(req *http.Request) (string, bool) {
	if req.Method != http.MethodPost || req.URL == nil {
		return "", false
	}
	operation := path.Clean(req.URL.Path)
	if _, ok := allowedOperations[operation]; !ok {
		return "", false
	}
	return operation, true
}

// requestPath returns the path of a request for logging, without assuming the
// request carries a URL.
func requestPath(req *http.Request) string {
	if req.URL == nil {
		return ""
	}
	return req.URL.Path
}

// NewServer returns a proxy server listening on the given address.
// Requests are forwarded to the endpoint in the given config.
// Requests are signed using credentials from the given config.
func NewServer(cfg *cfg.Config, awsCfg aws.Config) (*Server, error) {
	tcpAddr, err := net.ResolveTCPAddr("tcp", cfg.Socket.TCPAddress)
	if err != nil {
		log.Errorf("%v", err)
		os.Exit(1)
	}
	warnOnRemotelyReachableBind(tcpAddr)

	endPoint, er := getServiceEndpoint(&awsCfg)

	if er != nil {
		return nil, fmt.Errorf("%v", er)
	}

	log.Infof("HTTP Proxy server using X-Ray Endpoint : %v", endPoint)

	// Parse url from endpoint
	url, err := url.Parse(endPoint)
	if err != nil {
		return nil, fmt.Errorf("unable to parse xray endpoint: %v", err)
	}

	signer := v4.NewSigner()

	transport := conn.ProxyServerTransport(cfg)

	// Reverse proxy handler
	handler := &httputil.ReverseProxy{
		Transport: transport,

		// Handler for modifying and forwarding requests
		Director: func(req *http.Request) {
			if req != nil && req.URL != nil {
				log.Debugf("Received request on HTTP Proxy server : %s", req.URL.String())
			} else {
				log.Debug("Request/Request.URL received on HTTP Proxy server is nil")
			}

			// Remove connection header before signing request, otherwise the
			// reverse-proxy will remove the header before forwarding to X-Ray
			// resulting in a signed header being missing from the request.
			req.Header.Del(connHeader)

			// Set req url to xray endpoint
			req.URL.Scheme = url.Scheme
			req.URL.Host = url.Host
			req.Host = url.Host

			// Consume body and convert to io.ReadSeeker for signer to consume
			body, err := consume(req.Body)
			if err != nil {
				log.Errorf("Unable to consume request body: %v", err)

				// Forward unsigned request
				return
			}

			// Calculate payload hash
			// In SDK v2, we must manually calculate the payload hash for the SigV4 signer.
			// The v1 SDK's Sign() method handled this automatically, but v2's SignHTTP() requires
			// an explicit payloadHash parameter (hex-encoded SHA-256 of the request body).
			// Reference: https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/aws/signer/v4#Signer.SignHTTP
			var payloadHash string
			if body != nil {
				bodyBytes, _ := ioutil.ReadAll(body)
				hash := sha256.Sum256(bodyBytes)
				payloadHash = hex.EncodeToString(hash[:])
				// Reset body for request
				req.Body = ioutil.NopCloser(bytes.NewReader(bodyBytes))
				body = bytes.NewReader(bodyBytes)
			} else {
				hash := sha256.Sum256([]byte{})
				payloadHash = hex.EncodeToString(hash[:])
			}

			// Get credentials
			creds, err := awsCfg.Credentials.Retrieve(context.Background())
			if err != nil {
				log.Errorf("Unable to retrieve credentials: %v", err)
				return
			}

			// Sign request
			err = signer.SignHTTP(context.Background(), creds, req, payloadHash, service, awsCfg.Region, time.Now())
			if err != nil {
				log.Errorf("Unable to sign request: %v", err)
			}
		},
	}

	server := &http.Server{
		Addr:    cfg.Socket.TCPAddress,
		Handler: &operationFilter{proxy: handler},
	}

	p := &Server{server}

	return p, nil
}

// warnOnRemotelyReachableBind logs a warning when the proxy listens on anything
// other than a loopback address. The listener signs requests with the daemon's
// credentials and cannot authenticate its callers, so on such a bind every host
// that can route to the address may read the account's sampling rules using the
// daemon's IAM role.
func warnOnRemotelyReachableBind(addr *net.TCPAddr) {
	if addr.IP != nil && addr.IP.IsLoopback() {
		return
	}
	log.Warnf("HTTP Proxy server is bound to %v, which is reachable beyond the loopback interface. "+
		"The proxy signs requests with the daemon's credentials and cannot authenticate callers, so "+
		"restrict access to this address to the workloads that need it, for example with a security "+
		"group or a network policy.", addr.String())
}

// consume readsAll() the body and creates a new io.ReadSeeker from the content. v4.Signer
// requires an io.ReadSeeker to be able to sign requests. May return a nil io.ReadSeeker.
func consume(body io.ReadCloser) (io.ReadSeeker, error) {
	var buf []byte

	// Return nil ReadSeeker if body is nil
	if body == nil {
		return nil, nil
	}

	// Consume body
	buf, err := ioutil.ReadAll(body)
	if err != nil {
		return nil, err
	}

	return bytes.NewReader(buf), nil
}

// Serve starts server.
func (s *Server) Serve() {
	log.Infof("Starting proxy http server on %s", s.Addr)
	if err := s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Errorf("proxy http server failed to listen: %v", err)
	}
}

// Close stops server.
func (s *Server) Close() {
	err := s.Server.Close()
	if err != nil {
		log.Errorf("unable to close the server: %v", err)
	}
}

// getServiceEndpoint returns X-Ray service endpoint.
// It is guaranteed that awsCfg config instance is non-nil and the region value is non empty in awsCfg object.
// Currently the caller takes care of it.
func getServiceEndpoint(awsCfg *aws.Config) (string, error) {
	// Check for custom endpoint resolver (for testing)
	if awsCfg.EndpointResolverWithOptions != nil {
		ep, err := awsCfg.EndpointResolverWithOptions.ResolveEndpoint("xray", awsCfg.Region)
		if err == nil && ep.URL != "" {
			return ep.URL, nil
		}
	}
	
	if awsCfg.BaseEndpoint != nil && *awsCfg.BaseEndpoint != "" {
		return *awsCfg.BaseEndpoint, nil
	}
	
	if awsCfg.Region == "" {
		return "", errors.New("unable to generate endpoint from region with empty value")
	}
	
	// Generate X-Ray endpoint based on region partition
	var endpoint string
	
	// Handle special partitions
	if strings.HasPrefix(awsCfg.Region, "cn-") {
		// China regions
		endpoint = fmt.Sprintf("https://xray.%s.%s", awsCfg.Region, conn.DomainSuffixAWSCN)
	} else if strings.HasPrefix(awsCfg.Region, "us-iso-") {
		// ISO regions (US Isolated)
		endpoint = fmt.Sprintf("https://xray.%s.%s", awsCfg.Region, conn.DomainSuffixAWSISO)
	} else if strings.HasPrefix(awsCfg.Region, "us-isob-") {
		// ISO-B regions (US Isolated-B)
		endpoint = fmt.Sprintf("https://xray.%s.%s", awsCfg.Region, conn.DomainSuffixAWSISOB)
	} else {
		// Standard AWS regions (including GovCloud)
		endpoint = fmt.Sprintf("https://xray.%s.%s", awsCfg.Region, conn.DomainSuffixAWS)
	}
	
	return endpoint, nil
}
