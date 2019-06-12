// Copyright 2018-2018 Amazon.com, Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may not use this file except in compliance with the License. A copy of the License is located at
//
//     http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package ringbuffer

import (
	"math/rand"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/aws/aws-xray-daemon/daemon/bufferpool"
	"github.com/aws/aws-xray-daemon/daemon/telemetry"
	"github.com/aws/aws-xray-daemon/daemon/tracesegment"
	"github.com/aws/aws-xray-daemon/daemon/util/test"
	"github.com/stretchr/testify/assert"
)

func init() {
	telemetry.T = telemetry.GetTestTelemetry()
}

func TestRingBufferNewWithZeroCapacity(t *testing.T) {
	bufferLimit := 100
	bufferSize := 256 * 1024
	bufferPool := bufferpool.Init(bufferLimit, bufferSize)
	// Only run the failing part when a specific env variable is set
	if os.Getenv("Test_New") == "1" {
		New(0, bufferPool)
		return
	}
	// Start the actual test in a different subprocess
	cmd := exec.Command(os.Args[0], "-test.run=TestRingBufferNewWithZeroCapacity")
	cmd.Env = append(os.Environ(), "Test_New=1")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	// Check that the program exited
	err := cmd.Wait()
	if e, ok := err.(*exec.ExitError); !ok || e.Success() {
		t.Fatalf("Process ran with err %v, want exit status 1", err)
	}
}

func TestRingBufferNewWithDefaultCapacity(t *testing.T) {
	bufferLimit := 100
	bufferSize := 256 * 1024
	bufferPool := bufferpool.Init(bufferLimit, bufferSize)

	randomFlag := rand.Intn(defaultCapacity - 1)
	ringBuffer := New(randomFlag, bufferPool) // Ring buffer initialized with less than default capacity value

	assert.Equal(t, defaultCapacity, cap(ringBuffer.c), "The size of buffered channel should be equal to default capacity")
	assert.Equal(t, defaultCapacity, cap(ringBuffer.Channel), "The size of buffered channel should be equal to default capacity")
	assert.Equal(t, false, ringBuffer.Empty, "The ringBuffer is not empty")
	assert.Equal(t, uint64(0), ringBuffer.count, "The truncated count should be 0")
	assert.Equal(t, bufferPool, ringBuffer.pool, "The value of bufferpool should be same with the given value")

}

func TestRingBufferNew(t *testing.T) { // RingBuffer size greater than defaultCapacity
	bufferLimit := 100
	bufferSize := 256 * 1024
	bufferPool := bufferpool.Init(bufferLimit, bufferSize)

	randomFlag := getTestChannelSize()
	ringBuffer := New(randomFlag, bufferPool)

	assert.Equal(t, randomFlag, cap(ringBuffer.c), "The size of buffered channel should be same with the given number")
	assert.Equal(t, randomFlag, cap(ringBuffer.Channel), "The size of buffered channel should be same with the given number")
	assert.Equal(t, false, ringBuffer.Empty, "The ringBuffer is not empty")
	assert.Equal(t, uint64(0), ringBuffer.count, "The truncated count should be 0")
	assert.Equal(t, bufferPool, ringBuffer.pool, "The value of bufferpool should be same with the given value")

}

func TestRingBufferCloseChannel(t *testing.T) {
	bufferLimit := 100
	bufferSize := 256 * 1024
	bufferPool := bufferpool.Init(bufferLimit, bufferSize)
	randomFlag := getTestChannelSize()
	ringBuffer := New(randomFlag, bufferPool)
	ringBuffer.Close()
	for i := 0; i < cap(ringBuffer.c); i++ {
		v, ok := <-ringBuffer.c

		assert.Equal(t, (*tracesegment.TraceSegment)(nil), v, "The value should be nil")
		assert.Equal(t, false, ok, "The value should be false if the channel is closed")
	}
}

func TestRingBufferSend(t *testing.T) {
	bufferLimit := 100
	bufferSize := 256 * 1024
	bufferPool := bufferpool.Init(bufferLimit, bufferSize)
	randomFlag := getTestChannelSize()
	ringBuffer := New(randomFlag, bufferPool)
	segment := tracesegment.GetTestTraceSegment()
	for i := 0; i < randomFlag; i++ {
		ringBuffer.Send(&segment)
	}
	for i := 0; i < cap(ringBuffer.c); i++ {
		v, ok := <-ringBuffer.c

		assert.Equal(t, &segment, v, "The value should be same with the send segment")
		assert.Equal(t, true, ok, "The channel is open")
	}
}

func TestRingBufferTruncatedCount(t *testing.T) {
	log := test.LogSetup()
	bufferLimit := 100
	bufferSize := 256 * 1024
	bufferPool := bufferpool.Init(bufferLimit, bufferSize)
	segment := tracesegment.GetTestTraceSegment()
	randomFlag := getTestChannelSize()
	ringBuffer := New(randomFlag, bufferPool)
	extraSegments := 100
	for i := 0; i < randomFlag+extraSegments; i++ {
		ringBuffer.Send(&segment)
	}
	num := ringBuffer.TruncatedCount()

	assert.Equal(t, num, uint64(extraSegments), "The truncated count should be same with the extra segments sent")
	for i := 0; i < extraSegments; i++ {
		assert.True(t, strings.Contains(log.Logs[i], "Segment buffer is full. Dropping oldest segment document."))
	}
}

func TestRingBufferSendTruncated(t *testing.T) {
	log := test.LogSetup()
	bufferLimit := 100
	bufferSize := 256 * 1024
	bufferPool := bufferpool.Init(bufferLimit, bufferSize)
	randomFlag := getTestChannelSize() + 2
	ringBuffer := New(randomFlag, bufferPool)
	var segment []tracesegment.TraceSegment
	for i := 0; i < randomFlag; i++ {
		segment = append(segment, tracesegment.GetTestTraceSegment())
		ringBuffer.Send(&segment[i])
	}
	s1 := tracesegment.GetTestTraceSegment()
	ringBuffer.Send(&s1)

	assert.Equal(t, &segment[1], <-ringBuffer.c, "Truncate the first segment in the original buffered channel")
	assert.Equal(t, randomFlag, cap(ringBuffer.c), "The buffered channel still full after truncating")
	assert.True(t, strings.Contains(log.Logs[0], "Segment buffer is full. Dropping oldest segment document."))

	s2 := tracesegment.GetTestTraceSegment()
	ringBuffer.Send(&s2)

	assert.Equal(t, &segment[2], <-ringBuffer.c, "Truncate the second segment that in the original buffered channel")
	assert.Equal(t, randomFlag, cap(ringBuffer.c), "The buffered channel still full after truncating")
	assert.True(t, strings.Contains(log.Logs[0], "Segment buffer is full. Dropping oldest segment document."))
}

// getTestChannelSize returns a random number greater than or equal to defaultCapacity
func getTestChannelSize() int {
	return rand.Intn(50) + defaultCapacity
}
