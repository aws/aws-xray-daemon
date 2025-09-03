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
	"errors"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/xray"
	"github.com/aws/smithy-go"
	"github.com/aws/smithy-go/middleware"
	smithyhttp "github.com/aws/smithy-go/transport/http"
	daemoncfg "github.com/aws/aws-xray-daemon/pkg/cfg"
	log "github.com/cihub/seelog"
)

// Constant prefixes used to identify information in user-agent
const agentPrefix = "xray-agent/xray-daemon/"
const execEnvPrefix = " exec-env/"
const osPrefix = " OS/"

// XRay defines X-Ray api call structure.
type XRay interface {
	PutTraceSegments(ctx context.Context, input *xray.PutTraceSegmentsInput, opts ...func(*xray.Options)) (*xray.PutTraceSegmentsOutput, error)
	PutTelemetryRecords(ctx context.Context, input *xray.PutTelemetryRecordsInput, opts ...func(*xray.Options)) (*xray.PutTelemetryRecordsOutput, error)
}

// XRayClient represents X-Ray client.
type XRayClient struct {
	xRay *xray.Client
}

// PutTraceSegments makes PutTraceSegments api call on X-Ray client.
func (c *XRayClient) PutTraceSegments(ctx context.Context, input *xray.PutTraceSegmentsInput, opts ...func(*xray.Options)) (*xray.PutTraceSegmentsOutput, error) {
	return c.xRay.PutTraceSegments(ctx, input, opts...)
}

// PutTelemetryRecords makes PutTelemetryRecords api call on X-Ray client.
func (c *XRayClient) PutTelemetryRecords(ctx context.Context, input *xray.PutTelemetryRecordsInput, opts ...func(*xray.Options)) (*xray.PutTelemetryRecordsOutput, error) {
	return c.xRay.PutTelemetryRecords(ctx, input, opts...)
}

// NewXRay creates a new instance of the XRay client with aws configuration.
func NewXRay(cfg aws.Config) XRay {
	execEnv := os.Getenv("AWS_EXECUTION_ENV")
	if execEnv == "" {
		execEnv = "UNKNOWN"
	}

	osInformation := runtime.GOOS + "-" + runtime.GOARCH
	// User agent format: xray-daemon/3.x.x exec-env/ECS os/linux-amd64
	userAgent := agentPrefix + daemoncfg.Version + execEnvPrefix + execEnv + osPrefix + osInformation

	// Create X-Ray client with custom options
	x := xray.NewFromConfig(cfg, func(o *xray.Options) {
		o.APIOptions = append(o.APIOptions, func(stack *middleware.Stack) error {
			// Add user agent middleware
			return stack.Serialize.Add(middleware.SerializeMiddlewareFunc("XRayUserAgent", func(
				ctx context.Context, in middleware.SerializeInput, next middleware.SerializeHandler,
			) (middleware.SerializeOutput, middleware.Metadata, error) {
				req, ok := in.Request.(*smithyhttp.Request)
				if ok {
					existingUA := req.Header.Get("User-Agent")
					if existingUA != "" {
						req.Header.Set("User-Agent", existingUA+" "+userAgent)
					} else {
						req.Header.Set("User-Agent", userAgent)
					}
				}
				return next.HandleSerialize(ctx, in)
			}), middleware.After)
		})

		o.APIOptions = append(o.APIOptions, func(stack *middleware.Stack) error {
			// Add timestamp header middleware
			return stack.Finalize.Add(middleware.FinalizeMiddlewareFunc("XRayTimestamp", func(
				ctx context.Context, in middleware.FinalizeInput, next middleware.FinalizeHandler,
			) (middleware.FinalizeOutput, middleware.Metadata, error) {
				req, ok := in.Request.(*smithyhttp.Request)
				if ok {
					req.Header.Set("X-Amzn-Xray-Timestamp", strconv.FormatFloat(float64(time.Now().UnixNano())/float64(time.Second), 'f', 9, 64))
				}
				return next.HandleFinalize(ctx, in)
			}), middleware.Before)
		})
	})

	if cfg.BaseEndpoint != nil {
		log.Debugf("Using Endpoint: %s", *cfg.BaseEndpoint)
	} else {
		log.Debug("Using default X-Ray endpoint")
	}

	return &XRayClient{
		xRay: x,
	}
}

// IsTimeoutError checks whether error is timeout error.
func IsTimeoutError(err error) bool {
	if err == nil {
		return false
	}
	
	// Check for timeout errors
	// These string values are based on standard Go error messages and AWS SDK v2 timeout errors
	if strings.Contains(err.Error(), "request canceled") ||
		strings.Contains(err.Error(), "deadline exceeded") ||
		strings.Contains(err.Error(), "timeout") {
		return true
	}
	
	// Check for smithy operation errors
	var oe *smithy.OperationError
	if errors.As(err, &oe) {
		return IsTimeoutError(oe.Unwrap())
	}
	
	return false
}
