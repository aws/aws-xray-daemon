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
	"sync/atomic"
	"time"

	log "github.com/cihub/seelog"

	"github.com/aws/aws-xray-daemon/daemon/bufferpool"
	"github.com/aws/aws-xray-daemon/daemon/ringbuffer"
	"github.com/aws/aws-xray-daemon/daemon/tracesegment"

	"math/rand"
	"github.com/aws/aws-xray-daemon/daemon/cfg"
	"github.com/aws/aws-xray-daemon/daemon/conn"
	"github.com/aws/aws-xray-daemon/daemon/util/timer"
)

// Processor buffers segments and send to X-Ray service.
type Processor struct {
	// Boolean channel, set to true when processor has no segments in priority and standard ring buffer.
	Done chan bool

	// Ring buffer to store trace segments.
	std *ringbuffer.RingBuffer

	// Buffer pool instance.
	pool *bufferpool.BufferPool

	// Counter for segments received.
	count uint64

	// timer client used for setting idle timer.
	timerClient timer.Timer

	// segmentsBatch is used to process received segments batch.
	traceSegmentsBatch *segmentsBatch

	// Number of go routines to spawn for traceSegmentsBatch.poll().
	batchProcessorCount int

	// Channel for Time.
	idleTimer <-chan time.Time

	// Size of the batch segments processed by Processor.
	batchSize int

	// Idle timeout in milliseconds used while sending batch segments.
	sendIdleTimeout time.Duration
}

// New creates new instance of Processor.
func New(x conn.XRay, segmentBatchProcessorCount int, std *ringbuffer.RingBuffer,
	pool *bufferpool.BufferPool, c *cfg.ParameterConfig) *Processor {
	batchesChan := make(chan []*string, c.Processor.BatchProcessorQueueSize)
	segmentBatchDoneChan := make(chan bool)
	tsb := &segmentsBatch{
		batches: batchesChan,
		done:    segmentBatchDoneChan,
		randGen: rand.New(rand.NewSource(time.Now().UnixNano())),
		timer:   &timer.Client{},
	}
	tsb.xRay = x
	doneChan := make(chan bool)
	log.Debugf("Batch size: %v", c.Processor.BatchSize)
	p := &Processor{
		Done:                doneChan,
		std:                 std,
		pool:                pool,
		count:               0,
		timerClient:         &timer.Client{},
		batchProcessorCount: segmentBatchProcessorCount,
		traceSegmentsBatch:  tsb,
		batchSize:           c.Processor.BatchSize,
		sendIdleTimeout:     time.Millisecond * time.Duration(c.Processor.IdleTimeoutMillisecond),
	}

	for i := 0; i < p.batchProcessorCount; i++ {
		go p.traceSegmentsBatch.poll()
	}

	go p.poll()

	return p
}

func (p *Processor) poll() {
	batch := make([]*tracesegment.TraceSegment, 0, p.batchSize)
	p.SetIdleTimer()

	for {
		select {
		case segment, ok := <-p.std.Channel:
			if ok {
				batch = p.receiveTraceSegment(segment, batch)
			} else {
				p.std.Empty = true
			}
		case <-p.idleTimer:
			if len(batch) > 0 {
				log.Debug("processor: sending partial batch")
				batch = p.sendBatchAsync(batch)
			} else {
				p.SetIdleTimer()
			}
		}

		if p.std.Empty {
			break
		}
	}

	if len(batch) > 0 {
		batch = p.sendBatchAsync(batch)
	}
	p.traceSegmentsBatch.close()
	for i := 0; i < p.batchProcessorCount; i++ {
		<-p.traceSegmentsBatch.done
	}
	log.Debug("processor: done!")
	p.Done <- true
}

func (p *Processor) receiveTraceSegment(ts *tracesegment.TraceSegment, batch []*tracesegment.TraceSegment) []*tracesegment.TraceSegment {
	atomic.AddUint64(&p.count, 1)
	batch = append(batch, ts)

	if len(batch) >= p.batchSize {
		log.Debug("processor: sending complete batch")
		batch = p.sendBatchAsync(batch)
	} else if p.pool.CurrentBuffersLen() == 0 {
		log.Debug("processor: sending partial batch due to load on buffer pool")
		batch = p.sendBatchAsync(batch)
	}

	return batch
}

// Resizing slice doesn't make a copy of the underlying array and hence memory is not
// garbage collected. (http://blog.golang.org/go-slices-usage-and-internals)
func (p *Processor) flushBatch(batch []*tracesegment.TraceSegment) []*tracesegment.TraceSegment {
	for i := 0; i < len(batch); i++ {
		batch[i] = nil
	}
	batch = batch[0:0]

	return batch
}

func (p *Processor) sendBatchAsync(batch []*tracesegment.TraceSegment) []*tracesegment.TraceSegment {
	log.Debugf("processor: segment batch size: %d. capacity: %d", len(batch), cap(batch))

	segmentDocuments := []*string{}
	for _, segment := range batch {
		rawBytes := *segment.Raw
		x := string(rawBytes[:])
		segmentDocuments = append(segmentDocuments, &x)
		p.pool.Return(segment.PoolBuf)
	}
	p.traceSegmentsBatch.send(segmentDocuments)
	// Reset Idle Timer
	p.SetIdleTimer()
	return p.flushBatch(batch)
}

// ProcessedCount returns number of trace segment received.
func (p *Processor) ProcessedCount() uint64 {
	return atomic.LoadUint64(&p.count)
}

// SetIdleTimer sets idle timer for the processor instance.
func (p *Processor) SetIdleTimer() {
	p.idleTimer = p.timerClient.After(p.sendIdleTimeout)
}
