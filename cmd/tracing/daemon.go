// Copyright 2018-2018 Amazon.com, Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may not use this file except in compliance with the License. A copy of the License is located at
//
//     http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net"
	"os"
	"runtime/pprof"
	"sync/atomic"
	"time"

	"github.com/aws/aws-xray-daemon/pkg/bufferpool"
	"github.com/aws/aws-xray-daemon/pkg/cfg"
	"github.com/aws/aws-xray-daemon/pkg/cli"
	"github.com/aws/aws-xray-daemon/pkg/conn"
	"github.com/aws/aws-xray-daemon/pkg/logger"
	"github.com/aws/aws-xray-daemon/pkg/processor"
	"github.com/aws/aws-xray-daemon/pkg/profiler"
	"github.com/aws/aws-xray-daemon/pkg/proxy"
	"github.com/aws/aws-xray-daemon/pkg/ringbuffer"
	"github.com/aws/aws-xray-daemon/pkg/socketconn"
	"github.com/aws/aws-xray-daemon/pkg/socketconn/udp"
	"github.com/aws/aws-xray-daemon/pkg/telemetry"
	"github.com/aws/aws-xray-daemon/pkg/tracesegment"
	"github.com/aws/aws-xray-daemon/pkg/util"

	"github.com/aws/aws-sdk-go/aws"
	log "github.com/cihub/seelog"
	"github.com/shirou/gopsutil/mem"
)

var receiverCount int
var processorCount int
var config *cfg.Config

const protocolSeparator = "\n"

// Log Rotation Size is 50 MB
const logRotationSize int64 = 50 * 1024 * 1024

var udpAddress string
var tcpAddress string

var socketConnection string
var cpuProfile string
var memProfile string
var roleArn string
var receiveBufferSize int
var daemonProcessBufferMemoryMB int
var logFile string
var configFilePath string
var resourceARN string
var noMetadata bool
var version bool
var logLevel string
var regionFlag string
var proxyAddress string

// Daemon reads trace segments from X-Ray daemon address and
// send to X-Ray service.
type Daemon struct {
	// Boolean channel, set to true if error is received reading from Socket.
	done chan bool

	// Ring buffer, used to stored segments received.
	std *ringbuffer.RingBuffer

	// Counter for segments read by daemon.
	count uint64

	// Instance of socket connection.
	sock socketconn.SocketConn

	// Reference to buffer pool.
	pool *bufferpool.BufferPool

	// Reference to Processor.
	processor *processor.Processor

	// HTTP Proxy server
	server *proxy.Server
}

func init() {
	f, c := initCli("")
	f.ParseFlags()
	cfg.LogFile = logFile // storing log file passed through command line
	// if config file is passed using command line argument parse flags again with default equal to config file
	if configFilePath != "" {
		cfg.ConfigValidation(configFilePath)
		f, c = initCli(configFilePath)
		f.ParseFlags()
	}
	if version {
		fmt.Printf("AWS X-Ray daemon version: %v\n", cfg.Version)
		os.Exit(0)
	}
	config = c
	config.ProxyAddress = proxyAddress
}

func initCli(configFile string) (*cli.Flag, *cfg.Config) {
	flag := cli.NewFlag("X-Ray Daemon")
	cnfg := cfg.LoadConfig(configFile)
	processorCount = cnfg.Concurrency
	var (
		defaultDaemonProcessSpaceLimitMB = cnfg.TotalBufferSizeMB
		defaultLogPath                   = cnfg.Logging.LogPath
		defaultLogLevel                  = cnfg.Logging.LogLevel
		defaultUDPAddress                = cnfg.Socket.UDPAddress
		defaultTCPAddress                = cnfg.Socket.TCPAddress
		defaultRoleARN                   = cnfg.RoleARN
		defaultLocalMode                 = cnfg.LocalMode
		defaultRegion                    = cnfg.Region
		defaultResourceARN               = cnfg.ResourceARN
		defaultProxyAddress              = cnfg.ProxyAddress
	)
	socketConnection = "UDP"
	regionFlag = defaultRegion
	flag.StringVarF(&resourceARN, "resource-arn", "a", defaultResourceARN, "Amazon Resource Name (ARN) of the AWS resource running the daemon.")
	flag.BoolVarF(&noMetadata, "local-mode", "o", *defaultLocalMode, "Don't check for EC2 instance metadata.")
	flag.IntVarF(&daemonProcessBufferMemoryMB, "buffer-memory", "m", defaultDaemonProcessSpaceLimitMB, "Change the amount of memory in MB that buffers can use (minimum 3).")
	flag.StringVarF(&regionFlag, "region", "n", defaultRegion, "Send segments to X-Ray service in a specific region.")
	flag.StringVarF(&udpAddress, "bind", "b", defaultUDPAddress, "Overrides default UDP address (127.0.0.1:2000).")
	flag.StringVarF(&tcpAddress, "bind-tcp", "t", defaultTCPAddress, "Overrides default TCP address (127.0.0.1:2000).")
	flag.StringVarF(&roleArn, "role-arn", "r", defaultRoleARN, "Assume the specified IAM role to upload segments to a different account.")
	flag.StringVarF(&configFilePath, "config", "c", "", "Load a configuration file from the specified path.")
	flag.StringVarF(&logFile, "log-file", "f", defaultLogPath, "Output logs to the specified file path.")
	flag.StringVarF(&logLevel, "log-level", "l", defaultLogLevel, "Log level, from most verbose to least: dev, debug, info, warn, error, prod (default).")
	flag.StringVarF(&proxyAddress, "proxy-address", "p", defaultProxyAddress, "Proxy address through which to upload segments.")
	flag.BoolVarF(&version, "version", "v", false, "Show AWS X-Ray daemon version.")
	return flag, cnfg
}

func initDaemon(config *cfg.Config) *Daemon {
	if logFile != "" {
		var fileWriter io.Writer
		if *config.Logging.LogRotation {
			// Empty Archive path as code does not archive logs
			apath := ""
			maxSize := logRotationSize
			// Keep one rolled over log file around
			maxRolls := 1
			archiveExplode := false
			fileWriter, _ = log.NewRollingFileWriterSize(logFile, 0, apath, maxSize, maxRolls, 0, archiveExplode)
		} else {
			fileWriter, _ = log.NewFileWriter(logFile)
		}
		logger.LoadLogConfig(fileWriter, config, logLevel)
	} else {
		newWriter, _ := log.NewConsoleWriter()
		logger.LoadLogConfig(newWriter, config, logLevel)
	}
	defer log.Flush()

	log.Infof("Initializing AWS X-Ray daemon %v", cfg.Version)

	parameterConfig := cfg.ParameterConfigValue
	receiverCount = parameterConfig.ReceiverRoutines
	receiveBufferSize = parameterConfig.Socket.BufferSizeKB * 1024
	cpuProfile = os.Getenv("XRAY_DAEMON_CPU_PROFILE")
	memProfile = os.Getenv("XRAY_DAEMON_MEMORY_PROFILE")

	profiler.EnableCPUProfile(&cpuProfile)
	defer pprof.StopCPUProfile()

	var sock socketconn.SocketConn

	sock = udp.New(udpAddress)

	memoryLimit := evaluateBufferMemory(daemonProcessBufferMemoryMB)
	log.Infof("Using buffer memory limit of %v MB", memoryLimit)
	buffers, err := bufferpool.GetPoolBufferCount(memoryLimit, receiveBufferSize)
	if err != nil {
		log.Errorf("%v", err)
		os.Exit(1)
	}
	log.Infof("%v segment buffers allocated", buffers)
	bufferPool := bufferpool.Init(buffers, receiveBufferSize)
	std := ringbuffer.New(buffers, bufferPool)
	if config.Endpoint != "" {
		log.Debugf("Using Endpoint read from Config file: %s", config.Endpoint)
	}
	awsConfig, session := conn.GetAWSConfigSession(&conn.Conn{}, config, roleArn, regionFlag, noMetadata)
	log.Infof("Using region: %v", aws.StringValue(awsConfig.Region))

	log.Debugf("ARN of the AWS resource running the daemon: %v", resourceARN)
	telemetry.Init(awsConfig, session, resourceARN, noMetadata)

	// If calculated number of buffer is lower than our default, use calculated one. Otherwise, use default value.
	parameterConfig.Processor.BatchSize = util.GetMinIntValue(parameterConfig.Processor.BatchSize, buffers)

	config.Socket.TCPAddress = tcpAddress // assign final tcp address either through config file or cmd line
	// Create proxy http server
	server, err := proxy.NewServer(config, awsConfig, session)
	if err != nil {
		log.Errorf("Unable to start http proxy server: %v", err)
		os.Exit(1)
	}

	daemon := &Daemon{
		done:      make(chan bool),
		std:       std,
		pool:      bufferPool,
		count:     0,
		sock:      sock,
		server:    server,
		processor: processor.New(awsConfig, session, processorCount, std, bufferPool, parameterConfig),
	}

	return daemon
}

func runDaemon(daemon *Daemon) {
	// Start http server for proxying requests to xray
	go daemon.server.Serve()

	for i := 0; i < receiverCount; i++ {
		go daemon.poll()
	}
}

func (d *Daemon) close() {
	for i := 0; i < receiverCount; i++ {
		<-d.done
	}
	// Signal routines to finish
	// This will push telemetry and customer segments in parallel
	d.std.Close()
	telemetry.T.Quit <- true

	<-d.processor.Done
	<-telemetry.T.Done

	profiler.MemSnapShot(&memProfile)
	log.Debugf("Trace segment: received: %d, truncated: %d, processed: %d", atomic.LoadUint64(&d.count), d.std.TruncatedCount(), d.processor.ProcessedCount())
	log.Debugf("Shutdown finished. Current epoch in nanoseconds: %v", time.Now().UnixNano())
}

func (d *Daemon) stop() {
	d.sock.Close()
	d.server.Close()
}

// Returns number of bytes read from socket connection.
func (d *Daemon) read(buf *[]byte) int {
	bufVal := *buf
	rlen, err := d.sock.Read(bufVal)
	switch err := err.(type) {
	case net.Error:
		if !err.Temporary() {
			d.done <- true
			return -1
		}
		log.Errorf("daemon: net: err: %v", err)
		return 0
	case error:
		log.Errorf("daemon: socket: err: %v", err)
		return 0
	}
	return rlen
}

func (d *Daemon) poll() {
	separator := []byte(protocolSeparator)
	fallBackBuffer := make([]byte, receiveBufferSize)
	splitBuf := make([][]byte, 2)

	for {
		bufPointer := d.pool.Get()
		fallbackPointerUsed := false
		if bufPointer == nil {
			log.Debug("Pool does not have any buffer.")
			bufPointer = &fallBackBuffer
			fallbackPointerUsed = true
		}
		rlen := d.read(bufPointer)
		if rlen > 0 {
			telemetry.T.SegmentReceived(1)
		}
		if rlen == 0 {
			if !fallbackPointerUsed {
				d.pool.Return(bufPointer)
			}
			continue
		}
		if fallbackPointerUsed {
			log.Warn("Segment dropped. Consider increasing memory limit")
			telemetry.T.SegmentSpillover(1)
			continue
		} else if rlen == -1 {
			return
		}

		buf := *bufPointer
		bufMessage := buf[0:rlen]

		slices := util.SplitHeaderBody(&bufMessage, &separator, &splitBuf)
		if len(slices[1]) == 0 {
			log.Warnf("Missing header or segment: %s", string(slices[0]))
			d.pool.Return(bufPointer)
			telemetry.T.SegmentRejected(1)
			continue
		}

		header := slices[0]
		payload := slices[1]
		headerInfo := tracesegment.Header{}
		json.Unmarshal(header, &headerInfo)

		switch headerInfo.IsValid() {
		case true:
		default:
			log.Warnf("Invalid header: %s", string(header))
			d.pool.Return(bufPointer)
			telemetry.T.SegmentRejected(1)
			continue
		}

		ts := &tracesegment.TraceSegment{
			Raw:     &payload,
			PoolBuf: bufPointer,
		}

		atomic.AddUint64(&d.count, 1)
		d.std.Send(ts)
	}
}

func evaluateBufferMemory(cliBufferMemory int) int {
	var bufferMemoryMB int
	if cliBufferMemory > 0 {
		bufferMemoryMB = cliBufferMemory
	} else {
		vm, err := mem.VirtualMemory()
		if err != nil {
			log.Errorf("%v", err)
			os.Exit(1)
		}
		bufferMemoryLimitPercentageOfTotal := 0.01
		totalBytes := vm.Total
		bufferMemoryMB = int(math.Floor(bufferMemoryLimitPercentageOfTotal * float64(totalBytes) / float64(1024*1024)))
	}
	if bufferMemoryMB < 3 {
		log.Error("Not enough Buffers Memory Allocated. Min Buffers Memory required: 3 MB.")
		os.Exit(1)
	}
	return bufferMemoryMB
}
