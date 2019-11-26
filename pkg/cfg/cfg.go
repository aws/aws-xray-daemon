// Copyright 2018-2018 Amazon.com, Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may not use this file except in compliance with the License. A copy of the License is located at
//
//     http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package cfg

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"reflect"

	"github.com/aws/aws-xray-daemon/pkg/util"

	"gopkg.in/yaml.v2"

	log "github.com/cihub/seelog"
)

// Version number of the X-Ray daemon.
const Version = "3.1.0"

var cfgFileVersions = [...]int{1, 2} // Supported versions of cfg.yaml file.

var configLocations = []string{
	"/etc/amazon/xray/cfg.yaml",
	"cfg.yaml",
	"github.com/aws/aws-xray-daemon/pkg/cfg.yaml",
}

// LogFile represents log file passed through command line argument.
var LogFile string

// LogFormat defines format for logger.
var LogFormat = "%Date(2006-01-02T15:04:05Z07:00) [%Level] %Msg%n"

// Config defines configuration structure for cli parameters.
type Config struct {
	// Maximum buffer size in MB (minimum 3). Choose 0 to use 1% of host memory.
	TotalBufferSizeMB int `yaml:"TotalBufferSizeMB"`

	// Maximum number of concurrent calls to AWS X-Ray to upload segment documents.
	Concurrency int `yaml:"Concurrency"`

	// X-Ray service endpoint to which the daemon sends segment documents.
	Endpoint string `yaml:"Endpoint"`

	// Send segments to AWS X-Ray service in a specific region.
	Region string `yaml:"Region"`

	Socket struct {
		// Address and port on which the daemon listens for UDP packets containing segment documents.
		UDPAddress string `yaml:"UDPAddress"`
		TCPAddress string `yaml:"TCPAddress"`
	} `yaml:"Socket"`

	ProxyServer struct {
		IdleConnTimeout     int
		MaxIdleConnsPerHost int
		MaxIdleConns        int
	}

	// Structure for logging.
	Logging struct {
		// LogRotation, if true, will rotate log after 50 MB size of current log file.
		LogRotation *bool `yaml:"LogRotation"`
		// The log level, from most verbose to least: dev, debug, info, warn, error, prod (default).
		LogLevel string `yaml:"LogLevel"`
		// Logs to the specified file path.
		LogPath string `yaml:"LogPath"`
	} `yaml:"Logging"`

	// Local mode to skip EC2 instance metadata check.
	LocalMode *bool `yaml:"LocalMode"`

	// Amazon Resource Name (ARN) of the AWS resource running the daemon.
	ResourceARN string `yaml:"ResourceARN"`

	// IAM role to upload segments to a different account.
	RoleARN string `yaml:"RoleARN"`

	// Enable or disable TLS certificate verification.
	NoVerifySSL *bool `yaml:"NoVerifySSL"`

	// Upload segments to AWS X-Ray through a proxy.
	ProxyAddress string `yaml:"ProxyAddress"`

	// Daemon configuration file format version.
	Version int `yaml:"Version"`
}

// DefaultConfig returns default configuration for X-Ray daemon.
func DefaultConfig() *Config {
	return &Config{
		TotalBufferSizeMB: 0,
		Concurrency:       8,
		Endpoint:          "",
		Region:            "",
		Socket: struct {
			UDPAddress string `yaml:"UDPAddress"`
			TCPAddress string `yaml:"TCPAddress"`
		}{
			UDPAddress: "127.0.0.1:2000",
			TCPAddress: "127.0.0.1:2000",
		},
		ProxyServer: struct {
			IdleConnTimeout     int
			MaxIdleConnsPerHost int
			MaxIdleConns        int
		}{

			IdleConnTimeout:     30,
			MaxIdleConnsPerHost: 2,
			MaxIdleConns:        0,
		},
		Logging: struct {
			LogRotation *bool  `yaml:"LogRotation"`
			LogLevel    string `yaml:"LogLevel"`
			LogPath     string `yaml:"LogPath"`
		}{
			LogRotation: util.Bool(true),
			LogLevel:    "prod",
			LogPath:     "",
		},
		LocalMode:    util.Bool(false),
		ResourceARN:  "",
		RoleARN:      "",
		NoVerifySSL:  util.Bool(false),
		ProxyAddress: "",
		Version:      1,
	}
}

// ParameterConfig is a configuration used by daemon.
type ParameterConfig struct {
	SegmentChannel struct {
		// Size of trace segments channel.
		Std int
	}

	Socket struct {
		// Socket buffer size.
		BufferSizeKB int
	}

	// Number of go routines daemon.poll() to spawn.
	ReceiverRoutines int

	Processor struct {
		// Size of the batch segments processed by Processor.
		BatchSize int

		// Idle timeout in milliseconds used while sending batch segments.
		IdleTimeoutMillisecond int

		// MaxIdleConnPerHost, controls the maximum idle
		// (keep-alive) HTTP connections to keep per-host.
		MaxIdleConnPerHost int

		// Used to set Http client timeout in seconds.
		RequestTimeout          int
		BatchProcessorQueueSize int
	}
}

// ParameterConfigValue returns instance of ParameterConfig, initialized with default values.
var ParameterConfigValue = &ParameterConfig{
	SegmentChannel: struct {
		Std int
	}{
		Std: 250,
	},
	Socket: struct {
		BufferSizeKB int
	}{
		BufferSizeKB: 64,
	},
	ReceiverRoutines: 2,
	Processor: struct {
		BatchSize               int
		IdleTimeoutMillisecond  int
		MaxIdleConnPerHost      int
		RequestTimeout          int
		BatchProcessorQueueSize int
	}{
		BatchSize:               50,
		IdleTimeoutMillisecond:  1000,
		MaxIdleConnPerHost:      8,
		RequestTimeout:          2,
		BatchProcessorQueueSize: 20,
	},
}

// LoadConfig returns configuration from a valid configFile else default configuration.
func LoadConfig(configFile string) *Config {
	if configFile == "" {
		for _, val := range configLocations {
			if _, err := os.Stat(val); os.IsNotExist(err) {
				continue
			}
			return merge(val)
		}
		return DefaultConfig()
	}
	return merge(configFile)
}

func loadConfigFromFile(configPath string) *Config {
	bytes, err := ioutil.ReadFile(configPath)
	if err != nil {
		errorAndExit("", err)
	}
	return loadConfigFromBytes(bytes)
}

func loadConfigFromBytes(bytes []byte) *Config {
	c := &Config{}
	err := yaml.Unmarshal(bytes, c)
	if err != nil {
		errorAndExit("", err)
	}
	return c
}

func errorAndExit(serr string, err error) {
	createLogWritersAndLog(serr, err)
	rescueStderr := os.Stderr
	_, w, _ := os.Pipe()
	os.Stderr = w

	w.Close()
	os.Stderr = rescueStderr
	os.Exit(1)
}

// createLogWritersAndLog writes to stderr and provided log file.
func createLogWritersAndLog(serr string, err error) {
	var stderrWriter = os.Stderr
	var writer io.Writer

	stderrLogger, _ := log.LoggerFromWriterWithMinLevelAndFormat(stderrWriter, log.ErrorLvl, LogFormat)
	writeToLogger(stderrLogger, serr, err)

	if LogFile == "" {
		return
	}
	writer, _ = log.NewFileWriter(LogFile)
	fileLogger, _ := log.LoggerFromWriterWithMinLevelAndFormat(writer, log.ErrorLvl, LogFormat)
	writeToLogger(fileLogger, serr, err)
}

func writeToLogger(fileLogger log.LoggerInterface, serr string, err error) {
	log.ReplaceLogger(fileLogger)
	if serr != "" {
		log.Errorf("%v", serr)
	} else if err != nil {
		log.Errorf("Error occur when using config flag: %v", err)
	}
}

func configFlagArray(config yaml.MapSlice) []string {
	var configArray []string
	for i := 0; i < len(config); i++ {
		if config[i].Value == nil || reflect.TypeOf(config[i].Value).String() != "yaml.MapSlice" {
			configArray = append(configArray, fmt.Sprint(config[i].Key))
		} else {
			configItem := yaml.MapSlice{}
			configItem = config[i].Value.(yaml.MapSlice)
			for j := 0; j < len(configItem); j++ {
				configArray = append(configArray, fmt.Sprintf("%v.%v", config[i].Key, configItem[j].Key))
			}
		}
	}
	return configArray
}

func validConfigArray() []string {
	validConfig := yaml.MapSlice{}
	validConfigBytes, verr := yaml.Marshal(DefaultConfig())
	if verr != nil {
		errorAndExit("", verr)
	}
	yerr := yaml.Unmarshal(validConfigBytes, &validConfig)
	if yerr != nil {
		errorAndExit("", yerr)
	}
	return configFlagArray(validConfig)
}

func userConfigArray(configPath string) []string {
	fileBytes, rerr := ioutil.ReadFile(configPath)
	if rerr != nil {
		errorAndExit("", rerr)
	}
	userConfig := yaml.MapSlice{}
	uerr := yaml.Unmarshal(fileBytes, &userConfig)
	if uerr != nil {
		errorAndExit("", uerr)
	}
	return configFlagArray(userConfig)
}

// ConfigValidation validates provided configuration file, invalid configuration will exit the process.
func ConfigValidation(configPath string) {
	validConfigArray := validConfigArray()
	userConfigArray := userConfigArray(configPath)

	notSupportFlag := []string{"Profile.CPU", "Profile.Memory", "Socket.BufferSizeKB", "Logging.LogFormat", "Processor.BatchProcessorQueueSize"}
	needMigrateFlag := []string{"LogRotation", "Processor.Region", "Processor.Endpoint", "Processor.Routine", "MemoryLimit"}
	for i := 0; i < len(userConfigArray); i++ {
		if !contains(userConfigArray, "Version") {
			errorAndExit("Config Version is missing. Use X-Ray Daemon Config Migration Script to update the config file. Please refer to AWS X-Ray Documentation for more information.", nil)
		}
		if !contains(validConfigArray, userConfigArray[i]) {
			if contains(notSupportFlag, userConfigArray[i]) {
				errorMessage := fmt.Sprintf("%v flag is not supported any more. Please refer to AWS X-Ray Documentation for more information.", userConfigArray[i])
				errorAndExit(errorMessage, nil)
			} else if contains(needMigrateFlag, userConfigArray[i]) {
				errorMessage := fmt.Sprintf("%v flag is not supported. Use X-Ray Daemon Config Migration Script to update the config file. Please refer to AWS X-Ray Documentation for more information.", userConfigArray[i])
				errorAndExit(errorMessage, nil)
			} else {
				errorMessage := fmt.Sprintf("%v flag is invalid. Please refer to AWS X-Ray Documentation for more information.", userConfigArray[i])
				errorAndExit(errorMessage, nil)
			}
		}
	}
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func merge(configFile string) *Config {
	userConfig := loadConfigFromFile(configFile)
	versionMatch := false
	for i := 0; i < len(cfgFileVersions); i++ {
		if cfgFileVersions[i] == userConfig.Version {
			versionMatch = true
			break
		}
	}

	if !versionMatch {
		errorAndExit("Config Version Setting is not correct. Use X-Ray Daemon Config Migration Script to update the config file. Please refer to AWS X-Ray Documentation for more information.", nil)
	}

	userConfig.Socket.UDPAddress = getStringValue(userConfig.Socket.UDPAddress, DefaultConfig().Socket.UDPAddress)
	userConfig.Socket.TCPAddress = getStringValue(userConfig.Socket.TCPAddress, DefaultConfig().Socket.TCPAddress)
	userConfig.ProxyServer.IdleConnTimeout = DefaultConfig().ProxyServer.IdleConnTimeout
	userConfig.ProxyServer.MaxIdleConnsPerHost = DefaultConfig().ProxyServer.MaxIdleConnsPerHost
	userConfig.ProxyServer.MaxIdleConns = DefaultConfig().ProxyServer.MaxIdleConns
	userConfig.TotalBufferSizeMB = getIntValue(userConfig.TotalBufferSizeMB, DefaultConfig().TotalBufferSizeMB)
	userConfig.ResourceARN = getStringValue(userConfig.ResourceARN, DefaultConfig().ResourceARN)
	userConfig.RoleARN = getStringValue(userConfig.RoleARN, DefaultConfig().RoleARN)
	userConfig.Concurrency = getIntValue(userConfig.Concurrency, DefaultConfig().Concurrency)
	userConfig.Endpoint = getStringValue(userConfig.Endpoint, DefaultConfig().Endpoint)
	userConfig.Region = getStringValue(userConfig.Region, DefaultConfig().Region)
	userConfig.Logging.LogRotation = getBoolValue(userConfig.Logging.LogRotation, DefaultConfig().Logging.LogRotation)
	userConfig.Logging.LogLevel = getStringValue(userConfig.Logging.LogLevel, DefaultConfig().Logging.LogLevel)
	userConfig.Logging.LogPath = getStringValue(userConfig.Logging.LogPath, DefaultConfig().Logging.LogPath)
	userConfig.NoVerifySSL = getBoolValue(userConfig.NoVerifySSL, DefaultConfig().NoVerifySSL)
	userConfig.LocalMode = getBoolValue(userConfig.LocalMode, DefaultConfig().LocalMode)
	userConfig.ProxyAddress = getStringValue(userConfig.ProxyAddress, DefaultConfig().ProxyAddress)
	return userConfig
}

func getStringValue(configValue string, defaultValue string) string {
	if configValue == "" {
		return defaultValue
	}
	return configValue
}

func getIntValue(configValue, defaultValue int) int {
	if configValue == 0 {
		return defaultValue
	}
	return configValue
}

func getBoolValue(configValue, defaultValue *bool) *bool {
	if configValue == nil {
		return defaultValue
	}
	return configValue
}
