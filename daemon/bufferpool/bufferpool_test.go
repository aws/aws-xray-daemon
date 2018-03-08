// Copyright 2018-2018 Amazon.com, Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may not use this file except in compliance with the License. A copy of the License is located at
//
//     http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package bufferpool

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

type bufferPoolTestCase struct {
	processorSizeMB int
	bufferSizeKB    int
}

func TestBufferPoolGet(t *testing.T) {
	testCases := []int{10, 200, 1000, 5000, 10000}
	for _, bufferLimit := range testCases {

		bufferSize := 256 * 1024

		bufferPool := Init(bufferLimit, bufferSize)

		// First Fetch
		buf := bufferPool.Get()

		assert.EqualValues(t, bufferPool.CurrentBuffersLen(), bufferLimit-1)
		assert.NotNil(t, buf)

		// Try to get all. Minus 1 due to fetch above
		for i := 0; i < bufferLimit-1; i++ {
			buf = bufferPool.Get()

			assert.EqualValues(t, bufferPool.CurrentBuffersLen(), bufferLimit-1-(i+1))
			assert.NotNil(t, buf)
		}

		// No more buffer left hence returned nil
		buf = bufferPool.Get()

		assert.Nil(t, buf)
		assert.EqualValues(t, bufferPool.CurrentBuffersLen(), 0)
	}
}

func TestBufferReturn(t *testing.T) {
	bufferLimit := 10
	bufferSize := 256 * 1024
	bufferPool := Init(bufferLimit, bufferSize)
	buf := make([]byte, bufferSize)

	bufferPool.Return(&buf)

	// This return should be rejected as pool is already full
	assert.EqualValues(t, bufferPool.CurrentBuffersLen(), bufferLimit)

	// Fetch one and return buffer
	bufferPool.Get()
	assert.EqualValues(t, bufferPool.CurrentBuffersLen(), bufferLimit-1)

	bufferPool.Return(&buf)
	assert.EqualValues(t, bufferPool.CurrentBuffersLen(), bufferLimit)

	// Fetch two and return same buffer returned before which should be rejected
	returnedBuf1 := bufferPool.Get()
	returnedBuf2 := bufferPool.Get()

	assert.NotNil(t, returnedBuf1)
	assert.NotNil(t, returnedBuf2)
	assert.EqualValues(t, bufferPool.CurrentBuffersLen(), bufferLimit-2)

	bufferPool.Return(returnedBuf1)
	bufferPool.Return(returnedBuf1)

	assert.EqualValues(t, bufferPool.CurrentBuffersLen(), bufferLimit-1)
}

func TestBufferGetMultipleRoutine(t *testing.T) {
	testCases := []int{100, 1000, 2132}
	for _, bufferLimit := range testCases {
		bufferSize := 256 * 1024
		routines := 5
		pool := Init(bufferLimit, bufferSize)

		routineFunc := func(c chan int, pool *BufferPool) {
			count := 0
			for {
				buf := pool.Get()
				if buf == nil {
					break
				}
				count++
			}
			c <- count
		}
		chans := make([]chan int, routines)
		for i := 0; i < routines; i++ {
			c := make(chan int)
			chans[i] = c
			go routineFunc(c, pool)
		}

		totalFetched := 0
		for i := 0; i < routines; i++ {
			bufFetched := <-chans[i]
			totalFetched += bufFetched
		}

		assert.EqualValues(t, bufferLimit, totalFetched)
		buf := pool.Get()
		assert.Nil(t, buf)
	}
}

func TestGetPoolBufferCount(t *testing.T) {
	testCases := []bufferPoolTestCase{
		{processorSizeMB: 100, bufferSizeKB: 256},
		{processorSizeMB: 16, bufferSizeKB: 125},
		{processorSizeMB: 16, bufferSizeKB: 256},
		{processorSizeMB: 250, bufferSizeKB: 512},
		{processorSizeMB: 5, bufferSizeKB: 50},
	}

	for _, testCase := range testCases {
		processSizeMB := testCase.processorSizeMB
		bufferSize := testCase.bufferSizeKB

		bufferCount, err := GetPoolBufferCount(processSizeMB, bufferSize)

		assert.Nil(t, err)
		expected := int(math.Floor(float64((processSizeMB * 1024 * 1024) / bufferSize)))
		assert.EqualValues(t, expected, bufferCount)
	}
}

func TestGetPoolBufferCountNegativeProcessorSize(t *testing.T) {
	bufferCount, err := GetPoolBufferCount(-123, 24512)

	assert.EqualValues(t, 0, bufferCount)
	assert.NotNil(t, err)
	assert.EqualValues(t, err.Error(), "process limit MB cannot be less than or equal to zero")
}

func TestGetPoolBufferCountNegativeBufferSize(t *testing.T) {
	bufferCount, err := GetPoolBufferCount(123, -24512)

	assert.EqualValues(t, 0, bufferCount)
	assert.NotNil(t, err)
	assert.EqualValues(t, err.Error(), "receive buffer size cannot be less than or equal to zero")
}
