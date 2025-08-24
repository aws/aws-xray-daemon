// Copyright 2018-2018 Amazon.com, Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may not use this file except in compliance with the License. A copy of the License is located at
//
//     http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package processor

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/xray"
	"github.com/aws/aws-sdk-go-v2/service/xray/types"
	"github.com/aws/aws-xray-daemon/pkg/util/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var doneMsg = "Segment batch: done!"

type MockXRayClient struct {
	mock.Mock
	CallNoToPutTraceSegments int
	input                    *xray.PutTraceSegmentsInput
}

func (c *MockXRayClient) PutTraceSegments(ctx context.Context, input *xray.PutTraceSegmentsInput, opts ...func(*xray.Options)) (*xray.PutTraceSegmentsOutput, error) {
	c.input = input
	c.CallNoToPutTraceSegments++
	args := c.Called(nil)
	errorStr := args.String(0)
	var err error
	output := &xray.PutTraceSegmentsOutput{}
	if errorStr == "Send unprocessed" {
		segmentID := "Test-Segment-Id-1242113"
		output.UnprocessedTraceSegments = append(output.UnprocessedTraceSegments, types.UnprocessedTraceSegment{Id: &segmentID})
	} else if errorStr == "Send Invalid" {
		output.UnprocessedTraceSegments = append(output.UnprocessedTraceSegments, types.UnprocessedTraceSegment{Id: nil})
	} else if errorStr != "" {
		err = errors.New(errorStr)
	}
	return output, err
}

func (c *MockXRayClient) PutTelemetryRecords(ctx context.Context, input *xray.PutTelemetryRecordsInput, opts ...func(*xray.Options)) (*xray.PutTelemetryRecordsOutput, error) {
	return nil, nil
}

func TestSendOneBatch(t *testing.T) {
	s := segmentsBatch{
		batches: make(chan []string, 1),
	}
	testMessage := "Test Message"
	batch := []string{testMessage}

	s.send(batch)

	returnedBatch := <-s.batches
	assert.EqualValues(t, len(returnedBatch), 1)

	batchString := returnedBatch[0]
	assert.EqualValues(t, batchString, testMessage)
}

func TestSendBatchChannelTruncate(t *testing.T) {
	log := test.LogSetup()
	s := segmentsBatch{
		batches: make(chan []string, 1),
	}
	testMessage := "Test Message"
	batch := []string{testMessage}
	testMessage2 := "Test Message 2"
	batch2 := []string{testMessage2}

	s.send(batch)
	s.send(batch2)

	returnedBatch := <-s.batches

	assert.EqualValues(t, len(returnedBatch), 1)
	assert.EqualValues(t, returnedBatch[0], testMessage2)
	assert.True(t, strings.Contains(log.Logs[0], "Spilling over"))
	assert.True(t, strings.Contains(log.Logs[1], "retrying batch"))
}

func TestPollSendSuccess(t *testing.T) {
	log := test.LogSetup()
	xRay := new(MockXRayClient)
	xRay.On("PutTraceSegments", nil).Return("").Once()
	s := segmentsBatch{
		batches: make(chan []string, 1),
		xRay:    xRay,
		done:    make(chan bool),
	}
	testMessage := "{\"id\":\"9472\""
	batch := []string{testMessage}
	s.send(batch)

	go s.poll()
	close(s.batches)
	<-s.done

	assert.EqualValues(t, xRay.CallNoToPutTraceSegments, 1)
	assert.True(t, strings.Contains(log.Logs[0], fmt.Sprintf("Successfully sent batch of %v", 1)))
	assert.True(t, strings.Contains(log.Logs[1], doneMsg))
}

func TestPutTraceSegmentsParameters(t *testing.T) {
	log := test.LogSetup()
	xRay := new(MockXRayClient)
	xRay.On("PutTraceSegments", nil).Return("").Once()

	s := segmentsBatch{
		batches: make(chan []string, 1),
		xRay:    xRay,
		done:    make(chan bool),
	}
	testMessage := "{\"id\":\"9472\""
	batch := []string{testMessage}
	s.send(batch)

	go s.poll()

	close(s.batches)
	<-s.done
	actualInput := xRay.input

	expectedInput := &xray.PutTraceSegmentsInput{
		TraceSegmentDocuments: batch,
	}

	assert.EqualValues(t, actualInput, expectedInput)
	assert.EqualValues(t, xRay.CallNoToPutTraceSegments, 1)
	assert.True(t, strings.Contains(log.Logs[0], fmt.Sprintf("Successfully sent batch of %v", 1)))
	assert.True(t, strings.Contains(log.Logs[1], doneMsg))
}

func TestPollSendReturnUnprocessed(t *testing.T) {
	log := test.LogSetup()
	xRay := new(MockXRayClient)
	xRay.On("PutTraceSegments", nil).Return("Send unprocessed").Once()
	s := segmentsBatch{
		batches: make(chan []string, 1),
		xRay:    xRay,
		done:    make(chan bool),
	}
	testMessage := "{\"id\":\"9472\""
	batch := []string{testMessage}
	s.send(batch)

	go s.poll()
	close(s.batches)
	<-s.done

	assert.EqualValues(t, xRay.CallNoToPutTraceSegments, 1)
	assert.True(t, strings.Contains(log.Logs[0], fmt.Sprintf("Sent batch of %v segments but had %v Unprocessed segments", 1, 1)))
	assert.True(t, strings.Contains(log.Logs[1], "Unprocessed segment"))
}

func TestPollSendReturnUnprocessedInvalid(t *testing.T) {
	log := test.LogSetup()
	xRay := new(MockXRayClient)
	xRay.On("PutTraceSegments", nil).Return("Send Invalid").Once()
	s := segmentsBatch{
		batches: make(chan []string, 1),
		xRay:    xRay,
		done:    make(chan bool),
	}
	testMessage := "{\"id\":\"9472\""
	batch := []string{testMessage}
	s.send(batch)

	go s.poll()
	close(s.batches)
	<-s.done

	assert.EqualValues(t, xRay.CallNoToPutTraceSegments, 1)
	assert.True(t, strings.Contains(log.Logs[0], fmt.Sprintf("Sent batch of %v segments but had %v Unprocessed segments", 1, 1)))
	assert.True(t, strings.Contains(log.Logs[1], "Received nil unprocessed segment id from X-Ray service"))
}
