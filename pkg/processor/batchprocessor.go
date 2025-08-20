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
	"math/rand"
	"regexp"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/xray"
	"github.com/aws/aws-xray-daemon/pkg/conn"
	"github.com/aws/aws-xray-daemon/pkg/telemetry"
	"github.com/aws/aws-xray-daemon/pkg/util/timer"
	log "github.com/cihub/seelog"
)

var /* const */ segIdRegexp = regexp.MustCompile(`\"id\":\"(.*?)\"`)
var /* const */ traceIdRegexp = regexp.MustCompile(`\"trace_id\":\"(.*?)\"`)

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
	ctx := context.Background()
	for {
		batch, ok := <-s.batches
		if ok {
			// Convert []*string to []string
			segments := make([]string, len(batch))
			for i, seg := range batch {
				if seg != nil {
					segments[i] = *seg
				}
			}
			params := &xray.PutTraceSegmentsInput{
				TraceSegmentDocuments: segments,
			}
			start := time.Now()
			// send segment to X-Ray service.
			r, err := s.xRay.PutTraceSegments(ctx, params)
			if err != nil {
				telemetry.EvaluateConnectionError(err)
				log.Errorf("Sending segment batch failed with: %v", err)
				continue
			} else {
				telemetry.T.SegmentSent(int64(len(batch)))
			}
			elapsed := time.Since(start)

			if len(r.UnprocessedTraceSegments) != 0 {
				log.Infof("Sent batch of %d segments but had %d Unprocessed segments (%1.3f seconds)", len(batch),
					len(r.UnprocessedTraceSegments), elapsed.Seconds())
				batchesMap := make(map[string]string)
				for i := 0; i < len(batch); i++ {
					segIdStrs := segIdRegexp.FindStringSubmatch(*batch[i])
					if len(segIdStrs) != 2 {
						log.Debugf("Failed to match \"id\" in segment: %v", *batch[i])
						continue
					}
					batchesMap[segIdStrs[1]] = *batch[i]
				}
				for _, unprocessedSegment := range r.UnprocessedTraceSegments {
					telemetry.T.SegmentRejected(1)
					// Print all segments since don't know which exact one is invalid.
					if unprocessedSegment.Id == nil {
						log.Debugf("Received nil unprocessed segment id from X-Ray service: %v", unprocessedSegment)
						log.Debugf("Content in this batch: %v", params)
						break
					}
					traceIdStrs := traceIdRegexp.FindStringSubmatch(batchesMap[*unprocessedSegment.Id])
					if len(traceIdStrs) != 2 {
						log.Errorf("Unprocessed segment: %v", unprocessedSegment)
					} else {
						log.Errorf("Unprocessed trace %v, segment: %v", traceIdStrs[1], unprocessedSegment)
					}
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
