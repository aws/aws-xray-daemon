package test

import (
	log "github.com/cihub/seelog"
)

// LogWriter defines structure for log writer.
type LogWriter struct {
	Logs []string
}

// Write writes p bytes to log writer.
func (sw *LogWriter) Write(p []byte) (n int, err error) {
	sw.Logs = append(sw.Logs, string(p))
	return len(p), nil
}

// LogSetup initializes log writer.
func LogSetup() *LogWriter {
	writer := &LogWriter{}
	logger, err := log.LoggerFromWriterWithMinLevelAndFormat(writer, log.TraceLvl, "%Ns [%Level] %Msg")
	if err != nil {
		panic(err)
	}
	log.ReplaceLogger(logger)
	return writer
}
