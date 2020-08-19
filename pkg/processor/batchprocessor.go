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
	"github.com/aws/aws-xray-daemon/pkg/conn"
	"github.com/aws/aws-xray-daemon/pkg/telemetry"
	"github.com/aws/aws-xray-daemon/pkg/util/timer"
	"math"
	"math/rand"
	"regexp"
	"time"

	"github.com/aws/aws-sdk-go/service/xray"
	log "github.com/cihub/seelog"
)

const (
	backoffCapSeconds  = 30
	backoffMinAttempts = 10
	backoffBaseSeconds = 1
)

// Structure for trace segments batch.
type segmentsBatch struct {
	// Boolean channel set to true when processing the batch segments is done.
	done chan bool

	// String slice of trace segments.
	batches chan []*string

	// Instance of XRay, used to send data to X-Ray service.
	xRay conn.XRay

	// Random generator, used for back off logic in case of exceptions.
	randGen *rand.Rand

	// Instance of timer.
	timer timer.Timer
}

func (s *segmentsBatch) send(batch []*string) {
	select {
	case s.batches <- batch:

	default:
		select {
		case batchTruncated := <-s.batches:
			telemetry.T.SegmentSpillover(int64(len(batchTruncated)))
			log.Warnf("Spilling over %v segments", len(batchTruncated))

		default:
			log.Debug("Segment batch: channel is de-queued")
		}
		log.Debug("Segment batch: retrying batch")
		s.send(batch)
	}
}

func (s *segmentsBatch) poll() {
	failedAttempt := 0
	for {
		batch, ok := <-s.batches
		if ok {
			params := &xray.PutTraceSegmentsInput{
				TraceSegmentDocuments: batch,
			}
			start := time.Now()
			// send segment to X-Ray service.
			r, err := s.xRay.PutTraceSegments(params)
			if err != nil {
				telemetry.EvaluateConnectionError(err)
				failedAttempt++
				backOffSeconds := s.backOff(failedAttempt)
				log.Errorf("Sending segment batch failed with: %v", err)
				log.Warnf("Delaying sending of additional batches by %v seconds", backOffSeconds)
				if backOffSeconds > 0 {
					<-s.timer.After(time.Second * time.Duration(backOffSeconds))
				}
				continue
			} else {
				failedAttempt = 0
				telemetry.T.SegmentSent(int64(len(batch)))
			}
			elapsed := time.Since(start)

			batchesMap := make(map[string]string)
			idRegexp := regexp.MustCompile(`\"id\":\"(.*?)\"`)
			for i := 0; i < len(batch); i++ {
				idStrs := idRegexp.FindStringSubmatch(*batch[i])
				if len(idStrs) != 2 {
					log.Debugf("Failed to match \"id\" in segment: ", *batch[i])
					continue
				}
				batchesMap[idStrs[1]] = *batch[i]
			}

			if len(r.UnprocessedTraceSegments) != 0 {
				log.Infof("Sent batch of %d segments but had %d Unprocessed segments (%1.3f seconds)", len(batch),
					len(r.UnprocessedTraceSegments), elapsed.Seconds())
				for _, unprocessedSegment := range r.UnprocessedTraceSegments {
					telemetry.T.SegmentRejected(1)
					log.Errorf("Unprocessed segment: %v", unprocessedSegment)
					log.Debugf(batchesMap[*unprocessedSegment.Id])
				}
			} else {
				log.Infof("Successfully sent batch of %d segments (%1.3f seconds)", len(batch), elapsed.Seconds())
			}
		} else {
			log.Trace("Segment batch: done!")
			s.done <- true
			break
		}
	}
}

func (s *segmentsBatch) close() {
	close(s.batches)
}

func min(x, y int32) int32 {
	if x < y {
		return x
	}
	return y
}

// Returns int32 number for Full Jitter Base
// If the computation result in value greater than Max Int31 it returns MAX Int31 value
func getValidJitterBase(backoffBase, attempt int) int32 {
	base := float64(backoffBase) * math.Pow(2, float64(attempt))
	var baseInt int32
	if base > float64(math.MaxInt32/2) {
		baseInt = math.MaxInt32 / 2
	} else {
		baseInt = int32(base)
	}
	return baseInt
}

func (s *segmentsBatch) backOff(attempt int) int32 {
	if attempt <= backoffMinAttempts {
		return 0
	}
	// Attempts to be considered for Jitter Backoff
	backoffAttempts := attempt - backoffMinAttempts
	// As per Full Jitter described in https://www.awsarchitectureblog.com/2015/03/backoff.html
	base := getValidJitterBase(backoffBaseSeconds, backoffAttempts)
	randomBackoff := s.randGen.Int31n(base)
	return min(backoffCapSeconds, randomBackoff)
}
