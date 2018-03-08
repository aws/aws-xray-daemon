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
	"github.com/aws/aws-xray-daemon/daemon/bufferpool"
	"github.com/aws/aws-xray-daemon/daemon/telemetry"
	"github.com/aws/aws-xray-daemon/daemon/tracesegment"
)

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
func New(size int, pool *bufferpool.BufferPool) *RingBuffer {
	if size == 0 {
		log.Error("The initial size of a queue should be larger than 0")
		os.Exit(1)
	}
	channel := make(chan *tracesegment.TraceSegment, size)

	return &RingBuffer{
		Channel: channel,
		c:       channel,
		Empty:   false,
		count:   0,
		pool:    pool,
	}
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
