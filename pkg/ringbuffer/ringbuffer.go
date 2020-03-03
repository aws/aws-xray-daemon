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
	log "github.com/cihub/seelog"

	"os"

	"github.com/aws/aws-xray-daemon/pkg/bufferpool"
	"github.com/aws/aws-xray-daemon/pkg/telemetry"
	"github.com/aws/aws-xray-daemon/pkg/tracesegment"
)

var defaultCapacity = 250

// RingBuffer is used to store trace segment received on X-Ray daemon address.
type RingBuffer struct {
	// Channel used to store trace segment received on X-Ray daemon address.
	Channel <-chan *tracesegment.TraceSegment
	c       chan *tracesegment.TraceSegment

	// Boolean, set to true of buffer is empty
	Empty bool

	// Counter for trace segments truncated.
	count uint64

	// Reference to BufferPool.
	pool *bufferpool.BufferPool
}

// New returns new instance of RingBuffer configured with  BufferPool pool.
func New(bufferCount int, pool *bufferpool.BufferPool) *RingBuffer {
	if bufferCount == 0 {
		log.Error("The initial size of a queue should be larger than 0")
		os.Exit(1)
	}
	capacity := getChannelSize(bufferCount)
	channel := make(chan *tracesegment.TraceSegment, capacity)

	return &RingBuffer{
		Channel: channel,
		c:       channel,
		Empty:   false,
		count:   0,
		pool:    pool,
	}
}

// getChannelSize returns the size of the channel used by RingBuffer
// Currently 1X times the total number of allocated buffers for the X-Ray daemon is returned.
// This is proportional to number of buffers, since the segments are dropped if no new buffer can be allocated.
// max(defaultCapacity, bufferCount) is returned by the function.
func getChannelSize(bufferCount int) int {
	capacity := 1 * bufferCount
	if capacity < defaultCapacity {
		return defaultCapacity
	}
	return capacity
}

// Send sends trace segment s to trace segment channel.
func (r *RingBuffer) Send(s *tracesegment.TraceSegment) {
	select {
	case r.c <- s:
	default:
		var segmentTruncated *tracesegment.TraceSegment
		select {
		case segmentTruncated = <-r.c:
			r.count++
			r.pool.Return(segmentTruncated.PoolBuf)
			log.Warn("Segment buffer is full. Dropping oldest segment document.")
			telemetry.T.SegmentSpillover(1)
		default:
			log.Debug("Buffers: channel was de-queued")
		}
		r.Send(s)
	}
}

// Close closes the RingBuffer.
func (r *RingBuffer) Close() {
	close(r.c)
}

// TruncatedCount returns trace segment truncated count.
func (r *RingBuffer) TruncatedCount() uint64 {
	return r.count
}
