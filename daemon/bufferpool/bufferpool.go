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
	"errors"
	"math"
	"sync"
)

// BufferPool is a  structure for storing trace segments.
type BufferPool struct {
	// Slice of byte slices to store trace segments.
	Buffers []*[]byte
	lock    sync.Mutex

	// Map to track available buffers in the pool.
	bufferHeadHash map[*byte]bool
}

// Init initializes new BufferPool with bufferLimit buffers, each of bufferSize.
func Init(bufferLimit int, bufferSize int) *BufferPool {
	bufferHeadHash := make(map[*byte]bool)
	bufferArray := make([]*[]byte, bufferLimit)
	for i := 0; i < bufferLimit; i++ {
		buf := make([]byte, bufferSize)
		bufferArray[i] = &buf
		bufferHeadHash[getBufferPointer(&buf)] = true
	}
	bufferPool := BufferPool{
		Buffers:        bufferArray,
		lock:           sync.Mutex{},
		bufferHeadHash: bufferHeadHash,
	}
	return &bufferPool
}

// Get returns available buffer of BufferPool b, nil if not any.
func (b *BufferPool) Get() *[]byte {
	b.lock.Lock()
	buffers := b.Buffers
	buffersLen := len(buffers)
	var buf *[]byte
	if buffersLen > 0 {
		buf = buffers[buffersLen-1]
		b.Buffers = buffers[:buffersLen-1]
		delete(b.bufferHeadHash, getBufferPointer(buf))
	}
	b.lock.Unlock()
	return buf
}

// Return adds buffer buf to BufferPool b.
func (b *BufferPool) Return(buf *[]byte) {
	b.lock.Lock()
	// Rejecting buffer if already in pool
	if b.isBufferAlreadyInPool(buf) {
		b.lock.Unlock()
		return
	}
	buffers := b.Buffers
	buffersCap := cap(buffers)
	buffersLen := len(buffers)
	if buffersLen < buffersCap {
		buffers = append(buffers, buf)
		b.Buffers = buffers
		b.bufferHeadHash[getBufferPointer(buf)] = true
	}
	b.lock.Unlock()
}

// CurrentBuffersLen returns length of buffers.
func (b *BufferPool) CurrentBuffersLen() int {
	b.lock.Lock()
	len := len(b.Buffers)
	b.lock.Unlock()
	return len
}

func getBufferPointer(buf *[]byte) *byte {
	bufVal := *buf
	// Using first element as pointer to the whole array as Go array is continuous array
	// This might fail if someone return slice of original buffer that was fetched
	return &bufVal[0]
}

func (b *BufferPool) isBufferAlreadyInPool(buf *[]byte) bool {
	bufPointer := getBufferPointer(buf)
	_, ok := b.bufferHeadHash[bufPointer]
	return ok
}

// GetPoolBufferCount returns number of buffers that can fit in the given buffer pool limit
// where each buffer is of size receiveBufferSize.
func GetPoolBufferCount(bufferPoolLimitMB int, receiveBufferSize int) (int, error) {
	if receiveBufferSize <= 0 {
		return 0, errors.New("receive buffer size cannot be less than or equal to zero")
	}
	if bufferPoolLimitMB <= 0 {
		return 0, errors.New("process limit MB cannot be less than or equal to zero")
	}
	processLimitBytes := bufferPoolLimitMB * 1024 * 1024
	return int(math.Floor(float64(processLimitBytes / receiveBufferSize))), nil
}
