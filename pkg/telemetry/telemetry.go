// Copyright 2018-2018 Amazon.com, Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may not use this file except in compliance with the License. A copy of the License is located at
//
//     http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package telemetry

import (
	"context"
	"errors"
	"io"
	"os"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/aws/aws-xray-daemon/pkg/conn"
	"github.com/aws/aws-xray-daemon/pkg/util/timer"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/ec2/imds"
	"github.com/aws/aws-sdk-go-v2/service/xray"
	"github.com/aws/aws-sdk-go-v2/service/xray/types"
	"github.com/aws/smithy-go"
	smithyhttp "github.com/aws/smithy-go/transport/http"
	log "github.com/cihub/seelog"
)

const dataCutoffIntervalSecs = 60
const bufferSize = 30
const requestSize = 10

// T is instance of Telemetry.
var T *Telemetry

// Telemetry is used to record X-Ray daemon health.
type Telemetry struct {
	// Instance of XRay.
	client conn.XRay
	timer  timer.Timer

	// Amazon Resource Name (ARN) of the AWS resource running the daemon.
	resourceARN string

	// Instance id of the EC2 instance running X-Ray daemon.
	instanceID string

	// Host name of the EC2 instance running X-Ray daemon.
	hostname string

	// Self pointer.
	currentRecord *types.TelemetryRecord

	// Timer channel.
	timerChan <-chan time.Time

	// Boolean channel, set to true when Quit channel is set to true.
	Done chan bool

	// Boolean channel, set to true when daemon is closed,
	Quit chan bool

	// Channel of TelemetryRecord used to send to X-Ray service.
	recordChan chan types.TelemetryRecord

	// When segment is received, postTelemetry is set to true,
	// indicating send telemetry data for the received segment.
	postTelemetry bool
}

// Init instantiates a new instance of Telemetry.
func Init(ctx context.Context, cfg aws.Config, resourceARN string, noMetadata bool) {
	T = newT(ctx, cfg, resourceARN, noMetadata)
	log.Debug("Telemetry initiated")
}

// EvaluateConnectionError processes error with respect to request failure status code.
func EvaluateConnectionError(err error) {
	var oe *smithy.OperationError
	if errors.As(err, &oe) {
		if httpErr, ok := oe.Err.(*smithyhttp.ResponseError); ok {
			statusCode := httpErr.Response.StatusCode
			if statusCode >= 500 && statusCode < 600 {
				T.Connection5xx(1)
			} else if statusCode >= 400 && statusCode < 500 {
				T.Connection4xx(1)
			} else {
				T.ConnectionOther(1)
			}
		}
	} else {
		if conn.IsTimeoutError(err) {
			T.ConnectionTimeout(1)
		} else {
			var ae smithy.APIError
			if errors.As(err, &ae) {
				if ae.ErrorCode() == "RequestError" {
					T.ConnectionUnknownHost(1)
				}
			} else {
				T.ConnectionOther(1)
			}
		}
	}
}

// GetTestTelemetry returns an empty telemetry record.
func GetTestTelemetry() *Telemetry {
	return &Telemetry{
		currentRecord: getEmptyTelemetryRecord(),
	}
}

// SegmentReceived increments SegmentsReceivedCount for the Telemetry record.
func (t *Telemetry) SegmentReceived(count int64) {
	atomic.AddInt32(t.currentRecord.SegmentsReceivedCount, int32(count))
	// Only send telemetry data when we receive any segment or else skip any telemetry data
	t.postTelemetry = true
}

// SegmentSent increments SegmentsSentCount for the Telemetry record.
func (t *Telemetry) SegmentSent(count int64) {
	atomic.AddInt32(t.currentRecord.SegmentsSentCount, int32(count))
}

// SegmentSpillover increments SegmentsSpilloverCount for the Telemetry record.
func (t *Telemetry) SegmentSpillover(count int64) {
	atomic.AddInt32(t.currentRecord.SegmentsSpilloverCount, int32(count))
}

// SegmentRejected increments SegmentsRejectedCount for the Telemetry record.
func (t *Telemetry) SegmentRejected(count int64) {
	atomic.AddInt32(t.currentRecord.SegmentsRejectedCount, int32(count))
}

// ConnectionTimeout increments TimeoutCount for the Telemetry record.
func (t *Telemetry) ConnectionTimeout(count int64) {
	atomic.AddInt32(t.currentRecord.BackendConnectionErrors.TimeoutCount, int32(count))
}

// ConnectionRefusal increments ConnectionRefusedCount for the Telemetry record.
func (t *Telemetry) ConnectionRefusal(count int64) {
	atomic.AddInt32(t.currentRecord.BackendConnectionErrors.ConnectionRefusedCount, int32(count))
}

// Connection4xx increments HTTPCode4XXCount for the Telemetry record.
func (t *Telemetry) Connection4xx(count int64) {
	atomic.AddInt32(t.currentRecord.BackendConnectionErrors.HTTPCode4XXCount, int32(count))
}

// Connection5xx increments HTTPCode5XXCount count for the Telemetry record.
func (t *Telemetry) Connection5xx(count int64) {
	atomic.AddInt32(t.currentRecord.BackendConnectionErrors.HTTPCode5XXCount, int32(count))
}

// ConnectionUnknownHost increments unknown host BackendConnectionErrors count for the Telemetry record.
func (t *Telemetry) ConnectionUnknownHost(count int64) {
	atomic.AddInt32(t.currentRecord.BackendConnectionErrors.UnknownHostCount, int32(count))
}

// ConnectionOther increments other BackendConnectionErrors count for the Telemetry record.
func (t *Telemetry) ConnectionOther(count int64) {
	atomic.AddInt32(t.currentRecord.BackendConnectionErrors.OtherCount, int32(count))
}

func newT(ctx context.Context, cfg aws.Config, resourceARN string, noMetadata bool) *Telemetry {
	timer := &timer.Client{}
	hostname := ""
	instanceID := ""

	var metadataClient *imds.Client
	if !noMetadata {
		metadataClient = imds.NewFromConfig(cfg)
	}

	hostnameEnv := os.Getenv("AWS_HOSTNAME")
	if hostnameEnv != "" {
		hostname = hostnameEnv
		log.Debugf("Fetch hostname %v from environment variables", hostnameEnv)
	} else if metadataClient != nil {
		output, err := metadataClient.GetMetadata(ctx, &imds.GetMetadataInput{
			Path: "hostname",
		})
		if err != nil {
			log.Debugf("Get hostname metadata failed: %s", err)
		} else {
			// Read the content from the response
			if output.Content != nil {
				data, _ := io.ReadAll(output.Content)
				hostname = string(data)
				log.Debugf("Using %s hostname for telemetry records", hostname)
			}
		}
	} else {
		log.Debug("No hostname set for telemetry records")
	}

	instanceIDEnv := os.Getenv("AWS_INSTANCE_ID")
	if instanceIDEnv != "" {
		instanceID = instanceIDEnv
		log.Debugf("Fetch instance ID %v from environment variables", instanceIDEnv)
	} else if metadataClient != nil {
		output, err := metadataClient.GetMetadata(ctx, &imds.GetMetadataInput{
			Path: "instance-id",
		})
		if err != nil {
			log.Errorf("Get instance id metadata failed: %s", err)
		} else {
			// Read the content from the response
			if output.Content != nil {
				data, _ := io.ReadAll(output.Content)
				instanceID = string(data)
				log.Debugf("Using %s Instance Id for Telemetry records", instanceID)
			}
		}
	} else {
		log.Debug("No Instance Id set for telemetry records")
	}
	record := getEmptyTelemetryRecord()
	t := &Telemetry{
		timer:         timer,
		resourceARN:   resourceARN,
		instanceID:    instanceID,
		hostname:      hostname,
		currentRecord: record,
		timerChan:     getDataCutoffDelay(timer),
		Done:          make(chan bool),
		Quit:          make(chan bool),
		recordChan:    make(chan types.TelemetryRecord, bufferSize),
		postTelemetry: false,
	}
	telemetryClient := conn.NewXRay(cfg)
	t.client = telemetryClient
	go t.pushData(ctx)
	return t
}

func getZeroInt32() *int32 {
	var zero int32
	zero = 0
	return &zero
}

func getEmptyTelemetryRecord() *types.TelemetryRecord {
	return &types.TelemetryRecord{
		SegmentsReceivedCount:  getZeroInt32(),
		SegmentsRejectedCount:  getZeroInt32(),
		SegmentsSentCount:      getZeroInt32(),
		SegmentsSpilloverCount: getZeroInt32(),
		BackendConnectionErrors: &types.BackendConnectionErrors{
			HTTPCode4XXCount:       getZeroInt32(),
			HTTPCode5XXCount:       getZeroInt32(),
			ConnectionRefusedCount: getZeroInt32(),
			OtherCount:             getZeroInt32(),
			TimeoutCount:           getZeroInt32(),
			UnknownHostCount:       getZeroInt32(),
		},
	}
}

func (t *Telemetry) pushData(ctx context.Context) {
	for {
		quit := false
		select {
		case <-t.Quit:
			quit = true
			break
		case <-t.timerChan:
		}
		emptyRecord := getEmptyTelemetryRecord()
		recordToReport := unsafe.Pointer(emptyRecord)
		recordToPushPointer := unsafe.Pointer(t.currentRecord)
		// Rotation Logic:
		// Swap current record to record to report.
		// Record to report is set to empty record which is set to current record
		t.currentRecord = (*types.TelemetryRecord)(atomic.SwapPointer(&recordToReport,
			recordToPushPointer))
		currentTime := time.Now()
		record := (*types.TelemetryRecord)(recordToReport)
		record.Timestamp = &currentTime
		t.add(*record)
		t.sendAll(ctx)
		if quit {
			close(t.recordChan)
			log.Debug("telemetry: done!")
			t.Done <- true
			break
		} else {
			t.timerChan = getDataCutoffDelay(t.timer)
		}
	}
}

func (t *Telemetry) add(record types.TelemetryRecord) {
	// Only send telemetry data when we receive first segment or else do not send any telemetry data.
	if t.postTelemetry {
		select {
		case t.recordChan <- record:
		default:
			select {
			case <-t.recordChan:
				log.Debug("Telemetry Buffers truncated")
				t.add(record)
			default:
				log.Debug("Telemetry Buffers dequeued")
			}
		}
	} else {
		log.Debug("Skipped telemetry data as no segments found")
	}
}

func (t *Telemetry) sendAll(ctx context.Context) {
	records := t.collectAllRecords()
	recordsNoSend, err := t.sendRecords(ctx, records)
	if err != nil {
		log.Debugf("Failed to send telemetry %v record(s). Re-queue records. %v", len(records), err)
		// There might be possibility that new records might be archived during re-queue records.
		// But as timer is set after records are send this will not happen
		for _, record := range recordsNoSend {
			t.add(record)
		}
	}
}

func (t *Telemetry) collectAllRecords() []types.TelemetryRecord {
	records := make([]types.TelemetryRecord, 0, bufferSize)
	var record types.TelemetryRecord
	done := false
	for !done {
		select {
		case record = <-t.recordChan:
			recordLen := len(records)
			if recordLen < bufferSize {
				records = append(records, record)
			}
		default:
			done = true
		}
	}
	return records
}

func (t *Telemetry) sendRecords(ctx context.Context, records []types.TelemetryRecord) ([]types.TelemetryRecord, error) {
	if len(records) > 0 {
		for i := 0; i < len(records); i = i + requestSize {
			endIndex := len(records)
			if endIndex > i+requestSize {
				endIndex = i + requestSize
			}
			recordsToSend := records[i:endIndex]
			input := xray.PutTelemetryRecordsInput{
				EC2InstanceId:    &t.instanceID,
				Hostname:         &t.hostname,
				ResourceARN:      &t.resourceARN,
				TelemetryRecords: recordsToSend,
			}
			_, err := t.client.PutTelemetryRecords(ctx, &input)
			if err != nil {
				EvaluateConnectionError(err)
				return records[i:], err
			}
		}
		log.Debugf("Send %v telemetry record(s)", len(records))
	}
	return nil, nil
}

func getDataCutoffDelay(timer timer.Timer) <-chan time.Time {
	return timer.After(time.Duration(time.Second * dataCutoffIntervalSecs))
}
