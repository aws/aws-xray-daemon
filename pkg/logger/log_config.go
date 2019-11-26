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
	"io"
	"github.com/aws/aws-xray-daemon/pkg/cfg"

	log "github.com/cihub/seelog"
)

// LoadLogConfig configures Logger.
func LoadLogConfig(writer io.Writer, c *cfg.Config, loglevel string) {
	var level log.LogLevel

	switch c.Logging.LogLevel {
	case "dev":
		level = log.TraceLvl
	case "debug":
		level = log.DebugLvl
	case "info":
		level = log.InfoLvl
	case "warn":
		level = log.WarnLvl
	case "error":
		level = log.ErrorLvl
	case "prod":
		level = log.InfoLvl
	}

	if loglevel != c.Logging.LogLevel {
		switch loglevel {
		case "dev":
			level = log.TraceLvl
		case "debug":
			level = log.DebugLvl
		case "info":
			level = log.InfoLvl
		case "warn":
			level = log.WarnLvl
		case "error":
			level = log.ErrorLvl
		case "prod":
			level = log.InfoLvl
		}
	}

	logger, _ := log.LoggerFromWriterWithMinLevelAndFormat(writer, level, cfg.LogFormat)
	log.ReplaceLogger(logger)
}
