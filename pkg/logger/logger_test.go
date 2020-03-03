// Copyright 2018-2018 Amazon.com, Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may not use this file except in compliance with the License. A copy of the License is located at
//
//     http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package logger

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	"github.com/aws/aws-xray-daemon/pkg/cfg"
	"github.com/aws/aws-xray-daemon/pkg/util"

	"github.com/cihub/seelog"
	"github.com/stretchr/testify/assert"
)

type TestCase struct {
	Level   seelog.LogLevel
	Message string
	Params  []interface{}
	Output  string
}

func generateTestCase(t *testing.T, level seelog.LogLevel, formatID string, message string, params ...interface{}) TestCase {
	testCase := TestCase{
		Level:   level,
		Message: message,
		Params:  params,
	}
	var levelStr string
	switch level {
	case seelog.ErrorLvl:
		levelStr = "Error"

	case seelog.InfoLvl:
		levelStr = "Info"

	case seelog.DebugLvl:
		levelStr = "Debug"

	case seelog.WarnLvl:
		levelStr = "Warn"

	case seelog.TraceLvl:
		levelStr = "Trace"

	case seelog.CriticalLvl:
		levelStr = "Critical"

	default:
		assert.Fail(t, "Unexpected log level", level)
	}

	msg := fmt.Sprintf(testCase.Message, testCase.Params...)
	testCase.Output = fmt.Sprintf("%s [%v] %v\n", time.Now().Format(formatID), levelStr, msg)
	return testCase
}

func TestLogger(t *testing.T) {
	var testCases []TestCase

	formatID := "2006-01-02T15:04:05Z07:00"
	for _, logLevel := range []seelog.LogLevel{seelog.DebugLvl, seelog.InfoLvl, seelog.ErrorLvl, seelog.WarnLvl, seelog.TraceLvl, seelog.CriticalLvl} {
		testCases = append(testCases, generateTestCase(t, logLevel, formatID, "(some message without parameters)"))
		testCases = append(testCases, generateTestCase(t, logLevel, formatID, "(some message with %v as param)", []interface{}{"|a param|"}))
	}

	for _, testCase := range testCases {
		testLogger(t, testCase)
	}
}

func testLogger(t *testing.T, testCase TestCase) {
	// create seelog logger that outputs to buffer
	var out bytes.Buffer
	config := &cfg.Config{
		Logging: struct {
			LogRotation *bool  `yaml:"LogRotation"`
			LogLevel    string `yaml:"LogLevel"`
			LogPath     string `yaml:"LogPath"`
		}{
			LogRotation: util.Bool(true),
			LogLevel:    "dev",
			LogPath:     "/var/tmp/xray.log",
		},
	}
	// call loadlogconfig method under test
	loglevel := "dev"
	LoadLogConfig(&out, config, loglevel)
	// exercise logger
	switch testCase.Level {
	case seelog.ErrorLvl:
		if len(testCase.Params) > 0 {
			seelog.Errorf(testCase.Message, testCase.Params...)
		} else {
			seelog.Error(testCase.Message)
		}

	case seelog.InfoLvl:
		if len(testCase.Params) > 0 {
			seelog.Infof(testCase.Message, testCase.Params...)
		} else {
			seelog.Info(testCase.Message)
		}

	case seelog.DebugLvl:
		if len(testCase.Params) > 0 {
			seelog.Debugf(testCase.Message, testCase.Params...)
		} else {
			seelog.Debug(testCase.Message)
		}

	case seelog.WarnLvl:
		if len(testCase.Params) > 0 {
			seelog.Warnf(testCase.Message, testCase.Params...)
		} else {
			seelog.Warn(testCase.Message)
		}

	case seelog.TraceLvl:
		if len(testCase.Params) > 0 {
			seelog.Tracef(testCase.Message, testCase.Params...)
		} else {
			seelog.Trace(testCase.Message)
		}

	case seelog.CriticalLvl:
		if len(testCase.Params) > 0 {
			seelog.Criticalf(testCase.Message, testCase.Params...)
		} else {
			seelog.Critical(testCase.Message)
		}

	default:
		assert.Fail(t, "Unexpected log level", testCase.Level)
	}
	seelog.Flush()

	// check result
	assert.Equal(t, testCase.Output, out.String())
}
