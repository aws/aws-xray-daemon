// Package proxy provides an http server to act as a signing proxy for SDKs calling AWS X-Ray APIs
package proxy

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/aws/signer/v4"
	"github.com/aws/aws-xray-daemon/pkg/cfg"
	"github.com/aws/aws-xray-daemon/pkg/conn"
	log "github.com/cihub/seelog"
)

const service = "xray"
const connHeader = "Connection"

// Server represents HTTP server.
type Server struct {
	*http.Server
}

// NewServer returns a proxy server listening on the given address.
// Requests are forwarded to the endpoint in the given config.
// Requests are signed using credentials from the given config.
func NewServer(cfg *cfg.Config, awsCfg *aws.Config, sess *session.Session) (*Server, error) {
	_, err := net.ResolveTCPAddr("tcp", cfg.Socket.TCPAddress)
	if err != nil {
		log.Errorf("%v", err)
		os.Exit(1)
	}
	endPoint, er := getServiceEndpoint(awsCfg)

	if er != nil {
		return nil, fmt.Errorf("%v", er)
	}

	log.Infof("HTTP Proxy server using X-Ray Endpoint : %v", endPoint)

	// Parse url from endpoint
	url, err := url.Parse(endPoint)
	if err != nil {
		return nil, fmt.Errorf("unable to parse xray endpoint: %v", err)
	}

	signer := &v4.Signer{
		Credentials: sess.Config.Credentials,
	}

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

			// Sign request. signer.Sign() also repopulates the request body.
			_, err = signer.Sign(req, body, service, *awsCfg.Region, time.Now())
			if err != nil {
				log.Errorf("Unable to sign request: %v", err)
			}
		},
	}

	server := &http.Server{
		Addr:    cfg.Socket.TCPAddress,
		Handler: handler,
	}

	p := &Server{server}

	return p, nil
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
	err := s.ListenAndServe()
	if err != nil {
		log.Infof("proxy http server has stopped listening: %v", err)
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
// It is guaranteed that awsCfg config instance is non-nil and the region value is non nil or non empty in awsCfg object.
// Currently the caller takes care of it.
func getServiceEndpoint(awsCfg *aws.Config) (string, error) {
	if awsCfg.Endpoint == nil || *awsCfg.Endpoint == "" {
		if awsCfg.Region == nil || *awsCfg.Region == "" {
			return "", errors.New("unable to generate endpoint from region with nil value")
		}
		resolved, err := endpoints.DefaultResolver().EndpointFor(service, *awsCfg.Region, setResolverConfig())
		if err != nil {
			return "", err
		}

		return resolved.URL, err
	}
	return *awsCfg.Endpoint, nil
}

func setResolverConfig() func(*endpoints.Options) {
	return func(p *endpoints.Options) {
		p.ResolveUnknownService = true
	}
}
