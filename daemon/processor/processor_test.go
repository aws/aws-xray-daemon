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
	"fmt"
	"strings"
	"testing"
	"time"
	"github.com/aws/aws-xray-daemon/daemon/bufferpool"
	"github.com/aws/aws-xray-daemon/daemon/ringbuffer"
	"github.com/aws/aws-xray-daemon/daemon/telemetry"
	"github.com/aws/aws-xray-daemon/daemon/tracesegment"
	"github.com/aws/aws-xray-daemon/daemon/util/test"

	"github.com/stretchr/testify/assert"
)

func init() {
	telemetry.T = telemetry.GetTestTelemetry()
}

func TestFlushBatch(t *testing.T) {
	variousTests := []int{0, 10, 100, 324}
	for _, testCase := range variousTests {
		processor := Processor{}
		segments := make([]*tracesegment.TraceSegment, testCase)
		for i := 0; i < testCase; i++ {
			segmentVal := tracesegment.GetTestTraceSegment()
			segments[i] = &segmentVal
		}

		segmentsFlushed := processor.flushBatch(segments)

		assert.Equal(t, len(segmentsFlushed), 0)
		assert.Equal(t, cap(segmentsFlushed), testCase)
		for _, segmentVal := range segmentsFlushed {
			assert.Nil(t, segmentVal)
		}
	}
}

func TestSendBatchSuccess(t *testing.T) {
	timer := test.MockTimerClient{}
	variousTests := []int{0, 50, 40}
	for _, testCase := range variousTests {
		writer := test.LogSetup()
		segments := make([]*tracesegment.TraceSegment, testCase)
		for i := 0; i < testCase; i++ {
			segmentVal := tracesegment.GetTestTraceSegment()
			segments[i] = &segmentVal
		}
		processor := Processor{
			pool:        bufferpool.Init(testCase+1, 100),
			timerClient: &timer,
			traceSegmentsBatch: &segmentsBatch{
				batches: make(chan []*string, 1),
			},
		}
		// Empty Pool
		for i := 0; i < testCase+1; i++ {
			processor.pool.Get()
		}
		assert.EqualValues(t, processor.pool.CurrentBuffersLen(), 0)

		returnedSegment := processor.sendBatchAsync(segments)

		assert.EqualValues(t, cap(returnedSegment), cap(segments))
		assert.EqualValues(t, len(returnedSegment), 0)
		for _, segmentVal := range returnedSegment {
			assert.Nil(t, segmentVal)
		}
		assert.True(t, strings.Contains(writer.Logs[0], fmt.Sprintf("segment batch size: %v", testCase)))
		select {
		case batch := <-processor.traceSegmentsBatch.batches:
			assert.NotNil(t, batch)
		default:
			assert.Fail(t, "Expected batch to be in batch channel")
		}
		// Asserting the buffer pool was returned
		assert.EqualValues(t, processor.pool.CurrentBuffersLen(), testCase)
	}
	timer.Dispose()
}

func TestPollingFewSegmentsExit(t *testing.T) {
	pool := bufferpool.Init(1, 100)
	stdChan := ringbuffer.New(20, pool)
	doneChan := make(chan bool)
	timer := &test.MockTimerClient{}
	writer := test.LogSetup()
	processor := &Processor{
		timerClient: timer,
		std:         stdChan,
		count:       0,
		Done:        doneChan,
		pool:        pool,
		traceSegmentsBatch: &segmentsBatch{
			batches: make(chan []*string, 1),
		},
		sendIdleTimeout: time.Second,
		batchSize:       50,
	}

	go processor.poll()

	// Increment for Send Batch to proceed
	timer.IncrementDuration(time.Duration(10))
	segment := tracesegment.GetTestTraceSegment()
	stdChan.Send(&segment)
	stdChan.Close()

	<-processor.Done

	assert.EqualValues(t, processor.ProcessedCount(), 1)
	assert.True(t, strings.Contains(writer.Logs[0], "segment batch size: 1"))
	assert.True(t, strings.Contains(writer.Logs[1], "processor: done!"))

	timer.Dispose()
}

func TestPollingFewSegmentsIdleTimeout(t *testing.T) {
	pool := bufferpool.Init(1, 100)
	stdChan := ringbuffer.New(20, pool)
	doneChan := make(chan bool)
	timer := &test.MockTimerClient{}

	writer := test.LogSetup()
	processor := &Processor{
		timerClient: timer,
		std:         stdChan,
		count:       0,
		Done:        doneChan,
		pool:        pool,
		traceSegmentsBatch: &segmentsBatch{
			batches: make(chan []*string, 1),
		},
		sendIdleTimeout: time.Second,
		batchSize:       50,
	}

	go processor.poll()

	// Sleep to process go routine initialization
	time.Sleep(time.Millisecond)
	// Adding segment to priChan
	segment := tracesegment.GetTestTraceSegment()
	stdChan.Send(&segment)
	// Sleep to see to it the chan is processed before timeout is triggered
	time.Sleep(time.Millisecond)
	// Trigger Ideal Timeout to trigger PutSegments
	timer.IncrementDuration(processor.sendIdleTimeout)
	time.Sleep(time.Millisecond)
	// Sleep so that time.After trigger batch send and not closing of the channel
	stdChan.Close()

	<-doneChan

	assert.True(t, strings.Contains(writer.Logs[0], "sending partial batch"))
	assert.True(t, strings.Contains(writer.Logs[1], "segment batch size: 1"))
	assert.True(t, strings.Contains(writer.Logs[2], "processor: done!"))

	timer.Dispose()
}

func TestPollingBatchBufferFull(t *testing.T) {
	batchSize := 50
	pool := bufferpool.Init(1, 100)
	// Setting stdChan to batchSize so that it does not spill over
	stdChan := ringbuffer.New(batchSize, pool)
	doneChan := make(chan bool)
	timer := &test.MockTimerClient{}

	writer := test.LogSetup()
	segmentProcessorCount := 1
	processor := &Processor{
		timerClient:         timer,
		std:                 stdChan,
		count:               0,
		Done:                doneChan,
		batchProcessorCount: segmentProcessorCount,
		pool:                pool,
		traceSegmentsBatch: &segmentsBatch{
			batches: make(chan []*string, 1),
			done:    make(chan bool),
		},
		batchSize: batchSize,
	}

	go processor.poll()

	for i := 0; i < batchSize; i++ {
		// Adding segment to priChan
		segment := tracesegment.GetTestTraceSegment()
		stdChan.Send(&segment)

	}
	stdChan.Close()
	processor.traceSegmentsBatch.done <- true

	<-doneChan

	assert.EqualValues(t, processor.ProcessedCount(), batchSize)
	assert.True(t, strings.Contains(writer.Logs[0], "sending complete batch"))
	assert.True(t, strings.Contains(writer.Logs[1], fmt.Sprintf("segment batch size: %v", batchSize)))
	assert.True(t, strings.Contains(writer.Logs[2], "processor: done!"))

	timer.Dispose()
}

func TestPollingBufferPoolExhaustedForcingSent(t *testing.T) {
	pool := bufferpool.Init(1, 100)
	batchSize := 50
	// Exhaust the buffer pool
	pool.Get()
	assert.EqualValues(t, pool.CurrentBuffersLen(), 0)
	stdChan := ringbuffer.New(batchSize, pool)
	doneChan := make(chan bool)
	timer := &test.MockTimerClient{}

	writer := test.LogSetup()
	segmentProcessorCount := 1
	processor := &Processor{
		timerClient:         timer,
		std:                 stdChan,
		count:               0,
		Done:                doneChan,
		batchProcessorCount: segmentProcessorCount,
		pool:                pool,
		traceSegmentsBatch: &segmentsBatch{
			batches: make(chan []*string, 1),
			done:    make(chan bool),
		},
		sendIdleTimeout: time.Second,
		batchSize:       batchSize,
	}

	go processor.poll()

	segment := tracesegment.GetTestTraceSegment()
	stdChan.Send(&segment)
	stdChan.Close()
	processor.traceSegmentsBatch.done <- true

	<-doneChan

	assert.EqualValues(t, processor.ProcessedCount(), 1)
	assert.True(t, strings.Contains(writer.Logs[0], "sending partial batch due to load on buffer pool"))
	assert.True(t, strings.Contains(writer.Logs[1], fmt.Sprintf("segment batch size: %v", 1)))
	assert.True(t, strings.Contains(writer.Logs[2], "processor: done!"))

	timer.Dispose()
}

func TestPollingIdleTimerIsInitiatedAfterElapseWithNoSegments(t *testing.T) {
	timer := &test.MockTimerClient{}
	pool := bufferpool.Init(1, 100)
	batchSize := 50
	stdChan := ringbuffer.New(batchSize, pool)
	processor := &Processor{
		Done:        make(chan bool),
		timerClient: timer,
		std:         stdChan,
		pool:        pool,
		traceSegmentsBatch: &segmentsBatch{
			batches: make(chan []*string, 1),
		},
		sendIdleTimeout: time.Second,
		batchSize:       batchSize,
	}

	go processor.poll()

	// Sleep for routine to be initiated
	time.Sleep(time.Millisecond)
	// Trigger Idle Timeout
	timer.IncrementDuration(processor.sendIdleTimeout)
	// sleep so that routine exist after timeout is tiggered
	time.Sleep(time.Millisecond)
	stdChan.Close()
	<-processor.Done

	// Called twice once at poll start and then after the timeout was triggered
	assert.EqualValues(t, timer.AfterCalledTimes(), 2)
	timer.Dispose()
}
