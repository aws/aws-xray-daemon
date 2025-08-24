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
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/feature/ec2/imds"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/aws/smithy-go"
	daemoncfg "github.com/aws/aws-xray-daemon/pkg/cfg"
	log "github.com/cihub/seelog"
	"golang.org/x/net/http2"
)

type connAttr interface {
	newAWSConfig(ctx context.Context, roleArn string, region string) (aws.Config, error)
	getEC2Region(ctx context.Context, cfg aws.Config) (string, error)
}

// Conn implements connAttr interface.
type Conn struct{}

func (c *Conn) getEC2Region(ctx context.Context, cfg aws.Config) (string, error) {
	client := imds.NewFromConfig(cfg)
	regionResp, err := client.GetRegion(ctx, &imds.GetRegionInput{})
	if err != nil {
		return "", err
	}
	return regionResp.Region, nil
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
		IdleConnTimeout:     90 * time.Second, // Should be longer than PutTelemetryRecords call frequency: 60 seconds
		Proxy:               http.ProxyURL(proxyURL),
	}

	// is not enabled by default as we configure TLSClientConfig for supporting SSL to data plane.
	// http2.ConfigureTransport will setup transport layer to use HTTP2
	h2transport, err := http2.ConfigureTransports(transport)
	if err != nil {
		log.Warnf("Failed to configure HTTP2 transport: %v", err)
	} else {
		// Adding timeout settings to the http2 transport to prevent bad tcp connection hanging the requests for too long
		// See: https://t.corp.amazon.com/P104567981
		// Doc: https://pkg.go.dev/golang.org/x/net/http2#Transport
		//  - ReadIdleTimeout is the time before a ping is sent when no frame has been received from a connection
		//  - PingTimeout is the time before the TCP connection being closed if a Ping response is not received
		// So in total, if a TCP connection goes bad, it would take the combined time before the TCP connection is closed
		h2transport.ReadIdleTimeout = 1 * time.Second
		h2transport.PingTimeout = 2 * time.Second
	}

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

func getRegionFromECSMetadata() string {
	var ecsMetadataEnabled string
	var metadataFilePath string
	var metadataFile []byte
	var dat map[string]interface{}
	var taskArn []string
	var err error
	var region string
	region = ""
	ecsMetadataEnabled = os.Getenv("ECS_ENABLE_CONTAINER_METADATA")
	ecsMetadataEnabled = strings.ToLower(ecsMetadataEnabled)
	if ecsMetadataEnabled == "true" {
		metadataFilePath = os.Getenv("ECS_CONTAINER_METADATA_FILE")
		metadataFile, err = ioutil.ReadFile(metadataFilePath)
		if err != nil {
			log.Errorf("Unable to open ECS metadata file: %v\n", err)
		} else {
			if err := json.Unmarshal(metadataFile, &dat); err != nil {
				log.Errorf("Unable to read ECS metadata file contents: %v", err)
			} else {
				taskArn = strings.Split(dat["TaskARN"].(string), ":")
				region = taskArn[3]
				log.Debugf("Fetch region %v from ECS metadata file", region)
			}
		}
	}
	return region
}

// GetAWSConfig returns AWS config instance.
func GetAWSConfig(ctx context.Context, cn connAttr, c *daemoncfg.Config, roleArn string, region string, noMetadata bool) (aws.Config, error) {
	var cfg aws.Config
	var err error
	var awsRegion string
	http := getNewHTTPClient(daemoncfg.ParameterConfigValue.Processor.MaxIdleConnPerHost, daemoncfg.ParameterConfigValue.Processor.RequestTimeout, *c.NoVerifySSL, c.ProxyAddress)
	regionEnv := os.Getenv("AWS_REGION")
	if region == "" && regionEnv != "" {
		awsRegion = regionEnv
		log.Debugf("Fetch region %v from environment variables", awsRegion)
	} else if region != "" {
		awsRegion = region
		log.Debugf("Fetch region %v from commandline/config file", awsRegion)
	} else if !noMetadata {
		awsRegion = getRegionFromECSMetadata()
		if awsRegion == "" {
			tempCfg, err := getDefaultConfig(ctx)
			if err == nil {
				awsRegion, err = cn.getEC2Region(ctx, tempCfg)
				if err != nil {
					log.Errorf("Unable to fetch region from EC2 metadata: %v\n", err)
				} else {
					log.Debugf("Fetch region %s from ec2 metadata", awsRegion)
				}
			} else {
				log.Errorf("Unable to get default config: %v", err)
			}
		}
	} else {
		tempCfg, err := getDefaultConfig(ctx)
		if err == nil {
			awsRegion = tempCfg.Region
			log.Debugf("Fetched region %s from config", awsRegion)
		} else {
			log.Errorf("Unable to get default config: %v", err)
		}
	}
	if awsRegion == "" {
		log.Errorf("Cannot fetch region variable from config file, environment variables, ecs metadata, or ec2 metadata. Use local-mode to use the local config region.")
		os.Exit(1)
	}
	cfg, err = cn.newAWSConfig(ctx, roleArn, awsRegion)
	if err != nil {
		log.Errorf("Error creating AWS config: %v", err)
		os.Exit(1)
	}

	// Apply custom settings
	cfg.Region = awsRegion
	cfg.RetryMaxAttempts = 2
	if c.Endpoint != "" {
		cfg.BaseEndpoint = aws.String(c.Endpoint)
	}
	cfg.HTTPClient = http

	return cfg, nil
}

// ProxyServerTransport configures HTTP transport for TCP Proxy Server.
func ProxyServerTransport(config *daemoncfg.Config) *http.Transport {
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

func (c *Conn) newAWSConfig(ctx context.Context, roleArn string, region string) (aws.Config, error) {
	if roleArn == "" {
		return getDefaultConfig(ctx)
	}

	// Load config with STS credentials
	cfg, err := getDefaultConfig(ctx)
	if err != nil {
		return aws.Config{}, err
	}

	// Use STS to assume role
	stsCreds := getSTSCreds(ctx, cfg, region, roleArn)
	cfg.Credentials = stsCreds
	return cfg, nil
}

// getSTSCreds gets STS credentials from regional endpoint. RegionDisabledException is received if the
// STS regional endpoint is disabled. In this case STS credentials are fetched from STS primary regional endpoint
// in the respective AWS partition.
func getSTSCreds(ctx context.Context, cfg aws.Config, region string, roleArn string) aws.CredentialsProvider {
	stsCred := getSTSCredsFromRegionEndpoint(ctx, cfg, region, roleArn)
	
	// Test if credentials work
	_, err := stsCred.Retrieve(ctx)
	if err != nil {
		var ae smithy.APIError
		if errors.As(err, &ae) {
			if ae.ErrorCode() == "RegionDisabledException" {
				log.Errorf("Region : %s - %s", region, ae.ErrorMessage())
				log.Info("Credentials for provided RoleARN will be fetched from STS primary region endpoint instead of regional endpoint.")
				stsCred = getSTSCredsFromPrimaryRegionEndpoint(ctx, cfg, roleArn, region)
			}
		}
	}
	return stsCred
}

// getSTSCredsFromRegionEndpoint fetches STS credentials for provided roleARN from regional endpoint.
// AWS STS recommends that you provide both the Region and endpoint when you make calls to a Regional endpoint.
// Reference: https://docs.aws.amazon.com/IAM/latest/UserGuide/id_credentials_temp_enable-regions.html#id_credentials_temp_enable-regions_writing_code
func getSTSCredsFromRegionEndpoint(ctx context.Context, cfg aws.Config, region string, roleArn string) aws.CredentialsProvider {
	regionalEndpoint := getSTSRegionalEndpoint(region)
	// Configure STS client with regional endpoint
	cfg.Region = region
	if regionalEndpoint != "" {
		cfg.BaseEndpoint = aws.String(regionalEndpoint)
		log.Infof("Using STS regional endpoint: %s", regionalEndpoint)
	}
	stsClient := sts.NewFromConfig(cfg)
	log.Infof("STS Region : %s", region)
	return stscreds.NewAssumeRoleProvider(stsClient, roleArn)
}

// getSTSCredsFromPrimaryRegionEndpoint fetches STS credentials for provided roleARN from primary region endpoint in the
// respective partition.
func getSTSCredsFromPrimaryRegionEndpoint(ctx context.Context, cfg aws.Config, roleArn string, region string) aws.CredentialsProvider {
	partitionId := getPartition(region)
	if partitionId == PartitionAWS {
		return getSTSCredsFromRegionEndpoint(ctx, cfg, "us-east-1", roleArn)
	} else if partitionId == PartitionAWSCN {
		return getSTSCredsFromRegionEndpoint(ctx, cfg, "cn-north-1", roleArn)
	} else if partitionId == PartitionAWSUSGov {
		return getSTSCredsFromRegionEndpoint(ctx, cfg, "us-gov-west-1", roleArn)
	}

	return nil
}

func getSTSRegionalEndpoint(r string) string {
	p := getPartition(r)

	var e string
	if p == PartitionAWS || p == PartitionAWSUSGov {
		e = STSEndpointPrefix + r + STSEndpointSuffix
	} else if p == PartitionAWSCN {
		e = STSEndpointPrefix + r + STSAwsCnPartitionIDSuffix
	}
	return e
}

func getDefaultConfig(ctx context.Context) (aws.Config, error) {
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithEC2IMDSRegion(),
	)
	if err != nil {
		return aws.Config{}, err
	}
	return cfg, nil
}

// getPartition return AWS Partition for the provided region.
// Note: This method only returns correct results for Commercial/Gov/CN regions.
// ISO regions (aws-iso, aws-iso-b) are not supported and will default to "aws".
func getPartition(region string) string {
	// Simplified partition detection based on region prefixes
	if strings.HasPrefix(region, "cn-") {
		return PartitionAWSCN
	}
	if strings.HasPrefix(region, "us-gov-") {
		return PartitionAWSUSGov
	}
	// Check if it's a valid AWS region pattern
	if strings.Contains(region, "-") {
		return PartitionAWS
	}
	// Return empty for invalid regions
	return ""
}
