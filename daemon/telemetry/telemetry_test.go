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
	"errors"
	"fmt"
	"strings"
	"testing"
	"github.com/aws/aws-xray-daemon/daemon/util/test"

	"github.com/aws/aws-sdk-go/service/xray"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockXRayClient struct {
	mock.Mock
	CallNoToPutTelemetryRecords int
}

func (c *MockXRayClient) PutTraceSegments(input *xray.PutTraceSegmentsInput) (*xray.PutTraceSegmentsOutput, error) {
	return nil, nil
}

func (c *MockXRayClient) PutTelemetryRecords(input *xray.PutTelemetryRecordsInput) (*xray.PutTelemetryRecordsOutput, error) {
	c.CallNoToPutTelemetryRecords++
	args := c.Called(nil)
	errorStr := args.String(0)
	var err error
	output := &xray.PutTelemetryRecordsOutput{}
	if errorStr != "" {
		err = errors.New(errorStr)
	}
	return output, err
}

func TestGetEmptyTelemetryRecord(t *testing.T) {
	emptyRecord := getEmptyTelemetryRecord()

	assert.EqualValues(t, emptyRecord.SegmentsReceivedCount, new(int64))
	assert.EqualValues(t, emptyRecord.SegmentsRejectedCount, new(int64))
	assert.EqualValues(t, emptyRecord.SegmentsSentCount, new(int64))
	assert.EqualValues(t, emptyRecord.SegmentsSpilloverCount, new(int64))
	assert.EqualValues(t, emptyRecord.BackendConnectionErrors.ConnectionRefusedCount, new(int64))
	assert.EqualValues(t, emptyRecord.BackendConnectionErrors.HTTPCode4XXCount, new(int64))
	assert.EqualValues(t, emptyRecord.BackendConnectionErrors.HTTPCode5XXCount, new(int64))
	assert.EqualValues(t, emptyRecord.BackendConnectionErrors.OtherCount, new(int64))
	assert.EqualValues(t, emptyRecord.BackendConnectionErrors.TimeoutCount, new(int64))
	assert.EqualValues(t, emptyRecord.BackendConnectionErrors.UnknownHostCount, new(int64))
}

func TestAddTelemetryRecord(t *testing.T) {
	log := test.LogSetup()
	timer := &test.MockTimerClient{}
	telemetry := &Telemetry{
		client:        &MockXRayClient{},
		timer:         timer,
		resourceARN:   "",
		instanceID:    "",
		hostname:      "",
		currentRecord: getEmptyTelemetryRecord(),
		timerChan:     getDataCutoffDelay(timer),
		Done:          make(chan bool),
		Quit:          make(chan bool),
		recordChan:    make(chan *xray.TelemetryRecord, 1),
		postTelemetry: true,
	}

	telemetry.add(getEmptyTelemetryRecord())
	telemetry.add(getEmptyTelemetryRecord())

	assert.True(t, strings.Contains(log.Logs[0], "Telemetry Buffers truncated"))
}

func TestSendRecordSuccess(t *testing.T) {
	log := test.LogSetup()
	xRay := new(MockXRayClient)
	xRay.On("PutTelemetryRecords", nil).Return("").Once()
	timer := &test.MockTimerClient{}
	telemetry := &Telemetry{
		client:        xRay,
		timer:         timer,
		resourceARN:   "",
		instanceID:    "",
		hostname:      "",
		currentRecord: getEmptyTelemetryRecord(),
		timerChan:     getDataCutoffDelay(timer),
		Done:          make(chan bool),
		Quit:          make(chan bool),
		recordChan:    make(chan *xray.TelemetryRecord, 1),
	}
	records := make([]*xray.TelemetryRecord, 1)
	records[0] = getEmptyTelemetryRecord()
	telemetry.sendRecords(records)

	assert.EqualValues(t, xRay.CallNoToPutTelemetryRecords, 1)
	assert.True(t, strings.Contains(log.Logs[0], fmt.Sprintf("Send %v telemetry record(s)", 1)))
}

func TestAddRecordWithPostSegmentFalse(t *testing.T) {
	log := test.LogSetup()
	timer := &test.MockTimerClient{}
	telemetry := &Telemetry{
		client:        &MockXRayClient{},
		timer:         timer,
		resourceARN:   "",
		instanceID:    "",
		hostname:      "",
		currentRecord: getEmptyTelemetryRecord(),
		timerChan:     getDataCutoffDelay(timer),
		Done:          make(chan bool),
		Quit:          make(chan bool),
		recordChan:    make(chan *xray.TelemetryRecord, 1),
	}

	telemetry.add(getEmptyTelemetryRecord())

	assert.True(t, strings.Contains(log.Logs[0], "Skipped telemetry data as no segments found"))
}

func TestAddRecordBeforeFirstSegmentAndAfter(t *testing.T) {
	log := test.LogSetup()
	timer := &test.MockTimerClient{}
	telemetry := &Telemetry{
		client:        &MockXRayClient{},
		timer:         timer,
		resourceARN:   "",
		instanceID:    "",
		hostname:      "",
		currentRecord: getEmptyTelemetryRecord(),
		timerChan:     getDataCutoffDelay(timer),
		Done:          make(chan bool),
		Quit:          make(chan bool),
		recordChan:    make(chan *xray.TelemetryRecord, 1),
	}

	// No Segment received
	telemetry.add(getEmptyTelemetryRecord())

	assert.True(t, strings.Contains(log.Logs[0], "Skipped telemetry data as no segments found"))

	// Segment received
	telemetry.SegmentReceived(1)
	telemetry.add(getEmptyTelemetryRecord())
	telemetry.add(getEmptyTelemetryRecord())

	assert.True(t, strings.Contains(log.Logs[1], "Telemetry Buffers truncated"))
}
