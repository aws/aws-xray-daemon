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
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/service/xray"
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

func (c *MockXRayClient) PutTraceSegments(input *xray.PutTraceSegmentsInput) (*xray.PutTraceSegmentsOutput, error) {
	c.input = input
	c.CallNoToPutTraceSegments++
	args := c.Called(nil)
	errorStr := args.String(0)
	var err error
	output := &xray.PutTraceSegmentsOutput{}
	if errorStr == "Send unprocessed" {
		segmentID := "Test-Segment-Id-1242113"
		output.UnprocessedTraceSegments = append(output.UnprocessedTraceSegments, &xray.UnprocessedTraceSegment{Id: &segmentID})
	} else if errorStr == "Send Invalid" {
		output.UnprocessedTraceSegments = append(output.UnprocessedTraceSegments, &xray.UnprocessedTraceSegment{Id: nil})
	} else if errorStr != "" {
		err = errors.New(errorStr)
	}
	return output, err
}

func (c *MockXRayClient) PutTelemetryRecords(input *xray.PutTelemetryRecordsInput) (*xray.PutTelemetryRecordsOutput, error) {
	return nil, nil
}

func TestSendOneBatch(t *testing.T) {
	s := segmentsBatch{
		batches: make(chan []*string, 1),
	}
	testMessage := "Test Message"
	batch := []*string{&testMessage}

	s.send(batch)

	returnedBatch := <-s.batches
	assert.EqualValues(t, len(returnedBatch), 1)

	batchString := *returnedBatch[0]
	assert.EqualValues(t, batchString, testMessage)
}

func TestSendBatchChannelTruncate(t *testing.T) {
	log := test.LogSetup()
	s := segmentsBatch{
		batches: make(chan []*string, 1),
	}
	testMessage := "Test Message"
	batch := []*string{&testMessage}
	testMessage2 := "Test Message 2"
	batch2 := []*string{&testMessage2}

	s.send(batch)
	s.send(batch2)

	returnedBatch := <-s.batches

	assert.EqualValues(t, len(returnedBatch), 1)
	assert.EqualValues(t, *returnedBatch[0], testMessage2)
	assert.True(t, strings.Contains(log.Logs[0], "Spilling over"))
	assert.True(t, strings.Contains(log.Logs[1], "retrying batch"))
}

func TestPollSendSuccess(t *testing.T) {
	log := test.LogSetup()
	xRay := new(MockXRayClient)
	xRay.On("PutTraceSegments", nil).Return("").Once()
	s := segmentsBatch{
		batches: make(chan []*string, 1),
		xRay:    xRay,
		done:    make(chan bool),
	}
	testMessage := "{\"id\":\"9472\""
	batch := []*string{&testMessage}
	s.send(batch)

	go s.poll()
	close(s.batches)
	<-s.done

	assert.EqualValues(t, xRay.CallNoToPutTraceSegments, 1)
	assert.True(t, strings.Contains(log.Logs[0], fmt.Sprintf("Successfully sent batch of %v", 1)))
	assert.True(t, strings.Contains(log.Logs[1], doneMsg))
}

func TestPollSendFailedOnceMoreThanMin(t *testing.T) {
	seed := int64(122321)
	randGen := rand.New(rand.NewSource(seed))
	timer := test.MockTimerClient{}
	log := test.LogSetup()
	xRay := new(MockXRayClient)
	xRay.On("PutTraceSegments", nil).Return("Error")
	s := segmentsBatch{
		batches: make(chan []*string, 1),
		xRay:    xRay,
		done:    make(chan bool),
		randGen: rand.New(rand.NewSource(seed)),
		timer:   &timer,
	}
	testMessage := "Test Message"
	batch := []*string{&testMessage}
	// First failure
	backoff := randGen.Int31n(backoffBaseSeconds * 2)

	go s.poll()
	for i := 0; i < backoffMinAttempts; i++ {
		s.send(batch)
		timer.Advance(time.Second)
		time.Sleep(time.Millisecond)
	}
	s.send(batch)
	close(s.batches)

	time.Sleep(time.Millisecond)
	timer.Advance(time.Second * time.Duration(backoff))

	assert.EqualValues(t, xRay.CallNoToPutTraceSegments, backoffMinAttempts+1)
	// Backed off only once after min failed attempts are exhausted
	assert.EqualValues(t, 1, timer.AfterCalledTimes())

	<-s.done

	assert.True(t, strings.Contains(log.Logs[len(log.Logs)-1], doneMsg))
}

func TestPollSendFailedTwiceMoreThanMin(t *testing.T) {
	seed := int64(122321)
	randGen := rand.New(rand.NewSource(seed))
	timer := test.MockTimerClient{}
	log := test.LogSetup()
	xRay := new(MockXRayClient)
	xRay.On("PutTraceSegments", nil).Return("Error")
	s := segmentsBatch{
		batches: make(chan []*string, 1),
		xRay:    xRay,
		done:    make(chan bool),
		randGen: rand.New(rand.NewSource(seed)),
		timer:   &timer,
	}
	testMessage := "Test Message"
	batch := []*string{&testMessage}
	// First failure
	backoff := randGen.Int31n(backoffBaseSeconds * 2)

	go s.poll()
	for i := 0; i < backoffMinAttempts; i++ {
		s.send(batch)
		timer.Advance(time.Second)
		time.Sleep(time.Millisecond)
	}
	s.send(batch)

	time.Sleep(time.Millisecond)
	timer.Advance(time.Second * time.Duration(backoff))

	assert.EqualValues(t, xRay.CallNoToPutTraceSegments, backoffMinAttempts+1)
	assert.EqualValues(t, 1, timer.AfterCalledTimes())

	backoff2 := randGen.Int31n(backoffBaseSeconds * 4)

	s.send(batch)

	time.Sleep(time.Millisecond)
	timer.Advance(time.Second * time.Duration(backoff2))

	assert.EqualValues(t, xRay.CallNoToPutTraceSegments, backoffMinAttempts+2)
	assert.EqualValues(t, 2, timer.AfterCalledTimes())

	close(s.batches)
	<-s.done
	assert.True(t, strings.Contains(log.Logs[len(log.Logs)-1], doneMsg))
}

func TestPollSendFailedTwiceAndSucceedThird(t *testing.T) {
	seed := int64(122321)
	randGen := rand.New(rand.NewSource(seed))
	timer := test.MockTimerClient{}
	log := test.LogSetup()
	xRay := new(MockXRayClient)
	xRay.On("PutTraceSegments", nil).Return("Error").Times(backoffMinAttempts + 2)
	xRay.On("PutTraceSegments", nil).Return("").Once()

	s := segmentsBatch{
		batches: make(chan []*string, 1),
		xRay:    xRay,
		done:    make(chan bool),
		randGen: rand.New(rand.NewSource(seed)),
		timer:   &timer,
	}
	testMessage := "Test Message"
	batch := []*string{&testMessage}

	// First failure.
	backoff := randGen.Int31n(backoffBaseSeconds * 2)

	go s.poll()
	for i := 0; i < backoffMinAttempts; i++ {
		s.send(batch)
		timer.Advance(time.Second)
		time.Sleep(time.Millisecond)
	}
	s.send(batch)

	time.Sleep(time.Millisecond)
	timer.Advance(time.Second * time.Duration(backoff))

	assert.EqualValues(t, xRay.CallNoToPutTraceSegments, backoffMinAttempts+1)
	assert.EqualValues(t, 1, timer.AfterCalledTimes())

	// Second failure.
	backoff2 := randGen.Int31n(backoffBaseSeconds * 4)

	s.send(batch)

	time.Sleep(time.Millisecond)
	timer.Advance(time.Second * time.Duration(backoff2))

	assert.EqualValues(t, xRay.CallNoToPutTraceSegments, backoffMinAttempts+2)
	assert.EqualValues(t, 2, timer.AfterCalledTimes())

	// Third success.
	s.send(batch)

	time.Sleep(time.Millisecond)
	timer.Advance(time.Second)

	assert.EqualValues(t, xRay.CallNoToPutTraceSegments, backoffMinAttempts+3)
	assert.EqualValues(t, 2, timer.AfterCalledTimes()) // no backoff logic triggered.

	close(s.batches)
	<-s.done

	assert.True(t, strings.Contains(log.Logs[len(log.Logs)-2], fmt.Sprintf("Successfully sent batch of %v", 1)))
	assert.True(t, strings.Contains(log.Logs[len(log.Logs)-1], doneMsg))
}

func TestPutTraceSegmentsParameters(t *testing.T) {
	log := test.LogSetup()
	xRay := new(MockXRayClient)
	xRay.On("PutTraceSegments", nil).Return("").Once()

	s := segmentsBatch{
		batches: make(chan []*string, 1),
		xRay:    xRay,
		done:    make(chan bool),
	}
	testMessage := "{\"id\":\"9472\""
	batch := []*string{&testMessage}
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
		batches: make(chan []*string, 1),
		xRay:    xRay,
		done:    make(chan bool),
	}
	testMessage := "{\"id\":\"9472\""
	batch := []*string{&testMessage}
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
		batches: make(chan []*string, 1),
		xRay:    xRay,
		done:    make(chan bool),
	}
	testMessage := "{\"id\":\"9472\""
	batch := []*string{&testMessage}
	s.send(batch)

	go s.poll()
	close(s.batches)
	<-s.done

	assert.EqualValues(t, xRay.CallNoToPutTraceSegments, 1)
	assert.True(t, strings.Contains(log.Logs[0], fmt.Sprintf("Sent batch of %v segments but had %v Unprocessed segments", 1, 1)))
	assert.True(t, strings.Contains(log.Logs[1], "Received invalid unprocessed segment id from X-Ray"))
}

type minTestCase struct {
	x      int32
	y      int32
	result int32
}

func TestMin(t *testing.T) {
	testCases := []minTestCase{
		{x: 23, y: 54, result: 23},
		{x: 1121, y: 21, result: 21},
		{x: -12123, y: -4343, result: -12123},
		{x: 77, y: 77, result: 77},
		{x: 0, y: 0, result: 0},
		{x: 0, y: -54, result: -54},
		{x: -6543, y: 0, result: -6543},
	}
	for _, c := range testCases {
		r := min(c.x, c.y)

		assert.EqualValues(t, c.result, r, fmt.Sprintf("Min Test: X: %v, Y: %v, Expected: %v", c.x, c.y, c.result))
	}
}

func TestGetValidJitterBase(t *testing.T) {
	testCases := []struct {
		backoffBase   int
		attempt       int
		expectedValue int32
	}{
		{backoffBase: 1, attempt: 1, expectedValue: 2},
		{backoffBase: 2, attempt: 2, expectedValue: 8},
		{backoffBase: 1, attempt: 25, expectedValue: 33554432},
		{backoffBase: 5, attempt: 30, expectedValue: 1073741823},
		{backoffBase: 1, attempt: 100, expectedValue: 1073741823},
	}
	for _, tc := range testCases {
		backoffBase := tc.backoffBase
		attempt := tc.attempt

		base := getValidJitterBase(backoffBase, attempt)

		assert.EqualValues(t, tc.expectedValue, base)
	}
}

func TestBackoff(t *testing.T) {
	failedAttempts := []int{1, 2, 5, 7, 10, 23, 100, 1000, 343212}
	seedRandom := rand.New(rand.NewSource(time.Now().Unix()))
	for _, fa := range failedAttempts {
		seed := int64(seedRandom.Int63())
		randGen := rand.New(rand.NewSource(seed))
		s := segmentsBatch{
			randGen: rand.New(rand.NewSource(seed)),
		}

		backoffSec := s.backOff(fa)

		var backoffExpected int32

		if fa > backoffMinAttempts {
			randomBackoff := randGen.Int31n(getValidJitterBase(backoffBaseSeconds, fa-backoffMinAttempts))
			backoffExpected = randomBackoff
		}

		if backoffCapSeconds < backoffExpected {
			backoffExpected = backoffCapSeconds
		}
		assert.EqualValues(t, backoffExpected, backoffSec, fmt.Sprintf("Test Case: Failed Attempt: %v, Rand Seed: %v", fa, seed))
	}
}
