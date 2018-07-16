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
	newAWSSession(roleArn string) *session.Session
	getEC2Region(s *session.Session) (string, error)
}

// Conn implements connAttr interface.
type Conn struct{}

func (c *Conn) getEC2Region(s *session.Session) (string, error) {
	return ec2metadata.New(s).Region()
}

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
	s = cn.newAWSSession(roleArn)
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
	config := &aws.Config{
		Region:                 aws.String(awsRegion),
		DisableParamValidation: aws.Bool(true),
		MaxRetries:             aws.Int(2),
		Endpoint:               aws.String(c.Endpoint),
		HTTPClient:             http,
	}
	return config, s
}

func (c *Conn) newAWSSession(roleArn string) *session.Session {
	var s *session.Session
	var err error
	if roleArn == "" {
		s = getDefaultSession()
	} else {
		t := getDefaultSession()
		sts := stscreds.NewCredentialsWithClient(sts.New(t), roleArn)
		s, err = session.NewSession(&aws.Config{
			Credentials: sts,
		})

		if err != nil {
			log.Errorf("Error in creating session object : %v\n.", err)
			os.Exit(1)
		}
	}
	return s
}

func getDefaultSession() *session.Session {
	result, serr := session.NewSession()
	if serr != nil {
		log.Errorf("Error in creating session object : %v\n.", serr)
		os.Exit(1)
	}
	return result
}
