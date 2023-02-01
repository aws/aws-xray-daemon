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
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/xray"
	"github.com/aws/aws-xray-daemon/pkg/cfg"
	log "github.com/cihub/seelog"
)

// Constant prefixes used in to identify information in user-agent
const agentPrefix = "xray-agent/xray-daemon/"
const execEnvPrefix = " exec-env/"
const osPrefix = " OS/"

// XRay defines X-Ray api call structure.
type XRay interface {
	PutTraceSegments(input *xray.PutTraceSegmentsInput) (*xray.PutTraceSegmentsOutput, error)
	PutTelemetryRecords(input *xray.PutTelemetryRecordsInput) (*xray.PutTelemetryRecordsOutput, error)
}

// XRayClient represents X-Ray client.
type XRayClient struct {
	xRay *xray.XRay
}

// PutTraceSegments makes PutTraceSegments api call on X-Ray client.
func (c *XRayClient) PutTraceSegments(input *xray.PutTraceSegmentsInput) (*xray.PutTraceSegmentsOutput, error) {
	return c.xRay.PutTraceSegments(input)
}

// PutTelemetryRecords makes PutTelemetryRecords api call on X-Ray client.
func (c *XRayClient) PutTelemetryRecords(input *xray.PutTelemetryRecordsInput) (*xray.PutTelemetryRecordsOutput, error) {
	return c.xRay.PutTelemetryRecords(input)
}

// NewXRay creates a new instance of the XRay client with a aws configuration and session .
func NewXRay(awsConfig *aws.Config, s *session.Session) XRay {
	x := xray.New(s, awsConfig)
	log.Debugf("Using Endpoint: %s", x.Endpoint)

	execEnv := os.Getenv("AWS_EXECUTION_ENV")
	if execEnv == "" {
		execEnv = "UNKNOWN"
	}

	osInformation := runtime.GOOS + "-" + runtime.GOARCH

	x.Handlers.Build.PushBackNamed(request.NamedHandler{
		Name: "tracing.XRayVersionUserAgentHandler",
		Fn:   request.MakeAddToUserAgentFreeFormHandler(agentPrefix + cfg.Version + execEnvPrefix + execEnv + osPrefix + osInformation),
	})

	x.Handlers.Sign.PushFrontNamed(request.NamedHandler{
		Name: "tracing.TimestampHandler",
		Fn: func(r *request.Request) {
			r.HTTPRequest.Header.Set("X-Amzn-Xray-Timestamp", strconv.FormatFloat(float64(time.Now().UnixNano())/float64(time.Second), 'f', 9, 64))
		},
	})

	return &XRayClient{
		xRay: x,
	}
}

// IsTimeoutError checks whether error is timeout error.
func IsTimeoutError(err error) bool {
	awsError, ok := err.(awserr.Error)
	if ok {
		if strings.Contains(awsError.Error(), "net/http: request canceled") {
			return true
		}
	}
	return false
}
