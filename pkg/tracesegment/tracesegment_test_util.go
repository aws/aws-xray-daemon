// Copyright 2018-2018 Amazon.com, Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may not use this file except in compliance with the License. A copy of the License is located at
//
//     http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package tracesegment

import (
	"fmt"
	"math/rand"
)

// GetTestTraceSegment returns new instance of TraceSegment used for testing.
func GetTestTraceSegment() TraceSegment {
	traceRandomNumber := rand.Int()
	segmentRandomNumber := rand.Int()
	message := fmt.Sprintf("{\"trace_id\": \"%v\", \"id\": \"%v\", \"start_time\": 1461096053.37518, "+
		"\"end_time\": 1461096053.4042, "+
		"\"name\": \"hello-1.mbfzqxzcpe.us-east-1.elasticbeanstalk.com\"}",
		traceRandomNumber,
		segmentRandomNumber)
	buf := make([]byte, 100)
	messageBytes := []byte(message)

	segment := TraceSegment{
		PoolBuf: &buf,
		Raw:     &messageBytes,
	}
	return segment
}
