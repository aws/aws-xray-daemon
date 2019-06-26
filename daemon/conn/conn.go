// Copyright 2018-2018 Amazon.com, Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may not use this file except in compliance with the License. A copy of the License is located at
//
//     http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package conn

import (
	"crypto/tls"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-xray-daemon/daemon/cfg"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts"
	log "github.com/cihub/seelog"
	"golang.org/x/net/http2"
)

type connAttr interface {
	newAWSSession(roleArn string, region string) *session.Session
	getEC2Region(s *session.Session) (string, error)
}

// Conn implements connAttr interface.
type Conn struct{}

func (c *Conn) getEC2Region(s *session.Session) (string, error) {
	return ec2metadata.New(s).Region()
}

const (
	STSEndpointPrefix         = "https://sts."
	STSEndpointSuffix         = ".amazonaws.com"
	STSAwsCnPartitionIDSuffix = ".amazonaws.com.cn" // AWS China partition.
)

// getNewHTTPClient returns new HTTP client instance with provided configuration.
func getNewHTTPClient(maxIdle int, requestTimeout int, noVerify bool, proxyAddress string) *http.Client {
	log.Debugf("Using proxy address: %v", proxyAddress)
	tls := &tls.Config{
		InsecureSkipVerify: noVerify,
	}

	finalProxyAddress := getProxyAddress(proxyAddress)
	proxyURL := getProxyURL(finalProxyAddress)
	transport := &http.Transport{
		MaxIdleConnsPerHost: maxIdle,
		TLSClientConfig:     tls,
		Proxy:               http.ProxyURL(proxyURL),
	}

	// is not enabled by default as we configure TLSClientConfig for supporting SSL to data plane.
	// http2.ConfigureTransport will setup transport layer to use HTTP2
	http2.ConfigureTransport(transport)
	http := &http.Client{
		Transport: transport,
		Timeout:   time.Second * time.Duration(requestTimeout),
	}
	return http
}

func getProxyAddress(proxyAddress string) string {
	var finalProxyAddress string
	if proxyAddress != "" {
		finalProxyAddress = proxyAddress
	} else if proxyAddress == "" && os.Getenv("HTTPS_PROXY") != "" {
		finalProxyAddress = os.Getenv("HTTPS_PROXY")
	} else {
		finalProxyAddress = ""
	}
	return finalProxyAddress
}

func getProxyURL(finalProxyAddress string) *url.URL {
	var proxyURL *url.URL
	var err error
	if finalProxyAddress != "" {
		proxyURL, err = url.Parse(finalProxyAddress)
		if err != nil {
			log.Errorf("Bad proxy URL: %v", err)
			os.Exit(1)
		}
	} else {
		proxyURL = nil
	}
	return proxyURL
}

// GetAWSConfigSession returns AWS config and session instances.
func GetAWSConfigSession(cn connAttr, c *cfg.Config, roleArn string, region string, noMetadata bool) (*aws.Config, *session.Session) {
	var s *session.Session
	var err error
	var awsRegion string
	http := getNewHTTPClient(cfg.ParameterConfigValue.Processor.MaxIdleConnPerHost, cfg.ParameterConfigValue.Processor.RequestTimeout, *c.NoVerifySSL, c.ProxyAddress)
	regionEnv := os.Getenv("AWS_REGION")
	if region == "" && regionEnv != "" {
		awsRegion = regionEnv
		log.Debugf("Fetch region %v from environment variables", awsRegion)
	} else if region != "" {
		awsRegion = region
		log.Debugf("Fetch region %v from commandline/config file", awsRegion)
	} else if !noMetadata {
		es := getDefaultSession()
		awsRegion, err = cn.getEC2Region(es)
		if err != nil {
			log.Errorf("Unable to retrieve the region from the EC2 instance %v\n", err)
		} else {
			log.Debugf("Fetch region %v from ec2 metadata", awsRegion)
		}
	}
	if awsRegion == "" {
		log.Error("Cannot fetch region variable from config file, environment variables and ec2 metadata.")
		os.Exit(1)
	}
	s = cn.newAWSSession(roleArn, awsRegion)

	config := &aws.Config{
		Region:                 aws.String(awsRegion),
		DisableParamValidation: aws.Bool(true),
		MaxRetries:             aws.Int(2),
		Endpoint:               aws.String(c.Endpoint),
		HTTPClient:             http,
	}
	return config, s
}

// ProxyServerTransport configures HTTP transport for TCP Proxy Server.
func ProxyServerTransport(config *cfg.Config) *http.Transport {
	tls := &tls.Config{
		InsecureSkipVerify: *config.NoVerifySSL,
	}

	proxyAddr := getProxyAddress(config.ProxyAddress)
	proxyURL := getProxyURL(proxyAddr)

	// Connection timeout in seconds
	idleConnTimeout := time.Duration(config.ProxyServer.IdleConnTimeout) * time.Second

	transport := &http.Transport{
		MaxIdleConns:        config.ProxyServer.MaxIdleConns,
		MaxIdleConnsPerHost: config.ProxyServer.MaxIdleConnsPerHost,
		IdleConnTimeout:     idleConnTimeout,
		Proxy:               http.ProxyURL(proxyURL),
		TLSClientConfig:     tls,

		// If not disabled the transport will add a gzip encoding header
		// to requests with no `accept-encoding` header value. The header
		// is added after we sign the request which invalidates the
		// signature.
		DisableCompression: true,
	}

	return transport
}

func (c *Conn) newAWSSession(roleArn string, region string) *session.Session {
	var s *session.Session
	var err error
	if roleArn == "" {
		s = getDefaultSession()
	} else {
		stsCreds := getSTSCreds(region, roleArn)

		s, err = session.NewSession(&aws.Config{
			Credentials: stsCreds,
		})

		if err != nil {
			log.Errorf("Error in creating session object : %v\n.", err)
			os.Exit(1)
		}
	}
	return s
}

// getSTSCreds gets STS credentials from regional endpoint. ErrCodeRegionDisabledException is received if the
// STS regional endpoint is disabled. In this case STS credentials are fetched from STS primary regional endpoint
// in the respective AWS partition.
func getSTSCreds(region string, roleArn string) *credentials.Credentials {
	t := getDefaultSession()

	stsCred := getSTSCredsFromRegionEndpoint(t, region, roleArn)
	// Make explicit call to fetch credentials.
	_, err := stsCred.Get()
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case sts.ErrCodeRegionDisabledException:
				log.Errorf("Region : %v - %v", region, aerr.Error())
				log.Info("Credentials for provided RoleARN will be fetched from STS primary region endpoint instead of regional endpoint.")
				stsCred = getSTSCredsFromPrimaryRegionEndpoint(t, roleArn, region)
			}
		}
	}
	return stsCred
}

// getSTSCredsFromRegionEndpoint fetches STS credentials for provided roleARN from regional endpoint.
// AWS STS recommends that you provide both the Region and endpoint when you make calls to a Regional endpoint.
// Reference: https://docs.aws.amazon.com/IAM/latest/UserGuide/id_credentials_temp_enable-regions.html#id_credentials_temp_enable-regions_writing_code
func getSTSCredsFromRegionEndpoint(sess *session.Session, region string, roleArn string) *credentials.Credentials {
	regionalEndpoint := getSTSRegionalEndpoint(region)
	// if regionalEndpoint is "", the STS endpoint is Global endpoint for classic regions except ap-east-1 - (HKG)
	// for other opt-in regions, region value will create STS regional endpoint.
	// This will be only in the case, if provided region is not present in aws_regions.go
	c := &aws.Config{Region: aws.String(region), Endpoint: &regionalEndpoint}
	st := sts.New(sess, c)
	log.Infof("STS Endpoint : %v", st.Endpoint)
	return stscreds.NewCredentialsWithClient(st, roleArn)
}

// getSTSCredsFromPrimaryRegionEndpoint fetches STS credentials for provided roleARN from primary region endpoint in the
// respective partition.
func getSTSCredsFromPrimaryRegionEndpoint(t *session.Session, roleArn string, region string) *credentials.Credentials {
	partitionId := getPartition(region)
	if partitionId == endpoints.AwsPartitionID {
		return getSTSCredsFromRegionEndpoint(t, endpoints.UsEast1RegionID, roleArn)
	} else if partitionId == endpoints.AwsCnPartitionID {
		return getSTSCredsFromRegionEndpoint(t, endpoints.CnNorth1RegionID, roleArn)
	} else if partitionId == endpoints.AwsUsGovPartitionID {
		return getSTSCredsFromRegionEndpoint(t, endpoints.UsGovWest1RegionID, roleArn)
	}

	return nil
}

func getSTSRegionalEndpoint(r string) string {
	p := getPartition(r)

	var e string
	if p == endpoints.AwsPartitionID || p == endpoints.AwsUsGovPartitionID {
		e = STSEndpointPrefix + r + STSEndpointSuffix
	} else if p == endpoints.AwsCnPartitionID {
		e = STSEndpointPrefix + r + STSAwsCnPartitionIDSuffix
	}
	return e
}

func getDefaultSession() *session.Session {
	result, serr := session.NewSession()
	if serr != nil {
		log.Errorf("Error in creating session object : %v\n.", serr)
		os.Exit(1)
	}
	return result
}

// getPartition return AWS Partition for the provided region.
func getPartition(region string) string {
	p, _ := endpoints.PartitionForRegion(endpoints.DefaultPartitions(), region)
	return p.ID()
}
