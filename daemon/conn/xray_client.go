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
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/xray"
	log "github.com/cihub/seelog"
)

// Version number of the X-Ray daemon.
var versionNumber = "2.1.1"

// XRay defines X-Ray api call structure.
type XRay interface {
	PutTraceSegments(input *xray.PutTraceSegmentsInput) (*xray.PutTraceSegmentsOutput, error)
	PutTelemetryRecords(input *xray.PutTelemetryRecordsInput) (*xray.PutTelemetryRecordsOutput, error)
}

// XRayClient represents X-Ray client.
type XRayClient struct {
	xRay *xray.XRay
}

// GetVersionNumber returns version number of X-Ray daemon.
func GetVersionNumber() string {
	return versionNumber
}

// PutTraceSegments makes PutTraceSegments api call on X-Ray client.
func (c XRayClient) PutTraceSegments(input *xray.PutTraceSegmentsInput) (*xray.PutTraceSegmentsOutput, error) {
	return c.xRay.PutTraceSegments(input)
}

// PutTelemetryRecords makes PutTelemetryRecords api call on X-Ray client.
func (c XRayClient) PutTelemetryRecords(input *xray.PutTelemetryRecordsInput) (*xray.PutTelemetryRecordsOutput, error) {
	return c.xRay.PutTelemetryRecords(input)
}

// NewXRay creates a new instance of the XRay client with a aws configuration and session .
func NewXRay(awsConfig *aws.Config, s *session.Session) XRay {
	return requestXray(awsConfig, s)
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

func requestXray(awsConfig *aws.Config, s *session.Session) XRay {
	x := xray.New(s, awsConfig)
	log.Debugf("Using Endpoint: %s", x.Endpoint)
	var XRayVersionUserAgentHandler = request.NamedHandler{
		Name: "tracing.XRayVersionUserAgentHandler",
		Fn:   request.MakeAddToUserAgentHandler("xray", GetVersionNumber(), os.Getenv("AWS_EXECUTION_ENV")),
	}
	x.Handlers.Build.PushBackNamed(XRayVersionUserAgentHandler)

	xRay := XRayClient{
		xRay: x,
	}
	return xRay
}
