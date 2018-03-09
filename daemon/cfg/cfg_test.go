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
	"errors"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

var errFile = "error.log"
var tstFileName = "test_config.yaml"
var tstFilePath string

func setupTestCase() {
	LogFile = errFile
}

func tearTestCase() {
	LogFile = ""
	os.Remove(errFile)
}

func setupTestFile(cnfg string) (string, error) {
	goPath := os.Getenv("PWD")
	if goPath == "" {
		panic("GOPATH not set")
	}
	tstFilePath = goPath + "/" + tstFileName
	f, err := os.Create(tstFilePath)
	if err != nil {
		panic(err)
	}
	f.WriteString(cnfg)
	f.Close()
	return goPath, err
}

func clearTestFile() {
	os.Remove(tstFilePath)
}
func TestLoadConfigFromBytes(t *testing.T) {
	configString :=
		`Socket:
  UDPAddress: "127.0.0.1:2000"
TotalBufferSizeMB: 16
Region: "us-east-1"
Endpoint: "https://xxxx.xxxx.com"
ResourceARN: ""
RoleARN: ""
Concurrency: 8
Logging:
  LogRotation: true
  LogPath: ""
  LogLevel: "prod"
NoVerifySSL: false
LocalMode: false
ProxyAddress: ""
Version: 1`

	c := loadConfigFromBytes([]byte(configString))

	assert.EqualValues(t, c.Socket.UDPAddress, "127.0.0.1:2000")
	assert.EqualValues(t, c.TotalBufferSizeMB, 16)
	assert.EqualValues(t, c.Region, "us-east-1")
	assert.EqualValues(t, c.Endpoint, "https://xxxx.xxxx.com")
	assert.EqualValues(t, c.ResourceARN, "")
	assert.EqualValues(t, c.RoleARN, "")
	assert.EqualValues(t, c.Concurrency, 8)
	assert.EqualValues(t, c.Logging.LogLevel, "prod")
	assert.EqualValues(t, c.Logging.LogPath, "")
	assert.EqualValues(t, c.Logging.LogRotation, true)
	assert.EqualValues(t, c.NoVerifySSL, false)
	assert.EqualValues(t, c.LocalMode, false)
	assert.EqualValues(t, c.ProxyAddress, "")
	assert.EqualValues(t, c.Version, 1)
}

func TestLoadConfigFromBytesTypeError(t *testing.T) {
	configString :=
		`TotalBufferSizeMB: NotExist`

	// Only run the failing part when a specific env variable is set
	if os.Getenv("Test_Bytes") == "1" {
		loadConfigFromBytes([]byte(configString))
		return
	}

	// Start the actual test in a different subprocess
	cmd := exec.Command(os.Args[0], "-test.run=TestLoadConfigFromBytesTypeError")
	cmd.Env = append(os.Environ(), "Test_Bytes=1")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}

	// Check that the program exited
	err := cmd.Wait()
	if e, ok := err.(*exec.ExitError); !ok || e.Success() {
		t.Fatalf("Process ran with err %v, want exit status 1", err)
	}
}

func TestLoadConfigFromFile(t *testing.T) {
	configString :=
		`Socket:
  UDPAddress: "127.0.0.1:2000"
TotalBufferSizeMB: 16
Region: "us-east-1"
Endpoint: "https://xxxx.xxxx.com"
ResourceARN: ""
RoleARN: ""
Concurrency: 8
Logging:
  LogRotation: true
  LogPath: ""
  LogLevel: "prod"
NoVerifySSL: false
LocalMode: false
ProxyAddress: ""
Version: 1`
	setupTestFile(configString)

	c := loadConfigFromFile(tstFilePath)

	assert.EqualValues(t, c.Socket.UDPAddress, "127.0.0.1:2000")
	assert.EqualValues(t, c.TotalBufferSizeMB, 16)
	assert.EqualValues(t, c.Region, "us-east-1")
	assert.EqualValues(t, c.Endpoint, "https://xxxx.xxxx.com")
	assert.EqualValues(t, c.ResourceARN, "")
	assert.EqualValues(t, c.RoleARN, "")
	assert.EqualValues(t, c.Concurrency, 8)
	assert.EqualValues(t, c.Logging.LogLevel, "prod")
	assert.EqualValues(t, c.Logging.LogPath, "")
	assert.EqualValues(t, c.Logging.LogRotation, true)
	assert.EqualValues(t, c.NoVerifySSL, false)
	assert.EqualValues(t, c.LocalMode, false)
	assert.EqualValues(t, c.ProxyAddress, "")
	assert.EqualValues(t, c.Version, 1)

	clearTestFile()
}

func TestLoadConfigFromFileDoesNotExist(t *testing.T) {
	setupTestCase()
	testFile := "test_config_does_not_exist_121213.yaml"

	// Only run the failing part when a specific env variable is set
	if os.Getenv("Test_Bytes") == "1" {
		loadConfigFromFile(testFile)
		return
	}

	// Start the actual test in a different subprocess
	cmd := exec.Command(os.Args[0], "-test.run=TestLoadConfigFromFileDoesNotExist")
	cmd.Env = append(os.Environ(), "Test_Bytes=1")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}

	// Check that the program exited
	err := cmd.Wait()
	if e, ok := err.(*exec.ExitError); !ok || e.Success() {
		t.Fatalf("Process ran with err %v, want exit status 1", err)
	}
	tearTestCase()
}

func TestLoadConfig(t *testing.T) {
	configString :=
		`Socket:
  UDPAddress: "127.0.0.1:2000"
TotalBufferSizeMB: 16
Region: "us-east-1"
Endpoint: "https://xxxx.xxxx.com"
ResourceARN: ""
RoleARN: ""
Concurrency: 8
Logging:
  LogRotation: true
  LogPath: ""
  LogLevel: "prod"
NoVerifySSL: false
LocalMode: false
ProxyAddress: ""
Version: 1`
	setupTestFile(configString)
	configLocations = append([]string{tstFilePath}, configLocations...)

	c := LoadConfig("")

	assert.EqualValues(t, c.Socket.UDPAddress, "127.0.0.1:2000")
	assert.EqualValues(t, c.TotalBufferSizeMB, 16)
	assert.EqualValues(t, c.Region, "us-east-1")
	assert.EqualValues(t, c.Endpoint, "https://xxxx.xxxx.com")
	assert.EqualValues(t, c.ResourceARN, "")
	assert.EqualValues(t, c.RoleARN, "")
	assert.EqualValues(t, c.Concurrency, 8)
	assert.EqualValues(t, c.Logging.LogLevel, "prod")
	assert.EqualValues(t, c.Logging.LogPath, "")
	assert.EqualValues(t, c.Logging.LogRotation, true)
	assert.EqualValues(t, c.NoVerifySSL, false)
	assert.EqualValues(t, c.LocalMode, false)
	assert.EqualValues(t, c.ProxyAddress, "")
	assert.EqualValues(t, c.Version, 1)
	clearTestFile()
}

func TestLoadConfigFileNotPresent(t *testing.T) {
	configLocations = []string{"test_config_does_not_exist_989078070.yaml"}

	c := LoadConfig("")

	assert.NotNil(t, c)
	// If files config files are not present return default config
	assert.EqualValues(t, DefaultConfig(), c)
}

func TestMergeUserConfigWithDefaultConfig(t *testing.T) {
	configString :=
		`Socket:
  UDPAddress: "127.0.0.1:3000"
TotalBufferSizeMB: 8
Region: "us-east-2"
Endpoint: "https://xxxx.xxxx.com"
ResourceARN: ""
RoleARN: ""
Concurrency: 8
Version: 1`
	setupTestFile(configString)
	c := merge(tstFilePath)

	assert.EqualValues(t, c.Socket.UDPAddress, "127.0.0.1:3000")
	assert.EqualValues(t, c.TotalBufferSizeMB, 8)
	assert.EqualValues(t, c.Region, "us-east-2")
	assert.EqualValues(t, c.Endpoint, "https://xxxx.xxxx.com")
	assert.EqualValues(t, c.ResourceARN, "")
	assert.EqualValues(t, c.RoleARN, "")
	assert.EqualValues(t, c.Concurrency, 8)
	assert.EqualValues(t, c.Logging.LogLevel, "prod")
	assert.EqualValues(t, c.Logging.LogPath, "")
	assert.EqualValues(t, c.Logging.LogRotation, true)
	assert.EqualValues(t, c.NoVerifySSL, false)
	assert.EqualValues(t, c.LocalMode, false)
	assert.EqualValues(t, c.ProxyAddress, "")
	assert.EqualValues(t, c.Version, 1)
	clearTestFile()
}

func TestConfigVersionNotSet(t *testing.T) {
	setupTestCase()
	configString :=
		`Socket:
  UDPAddress: "127.0.0.1:3000"
TotalBufferSizeMB: 8
Region: "us-east-2"
Endpoint: "https://xxxx.xxxx.com"
ResourceARN: ""
RoleARN: ""
Concurrency: 8`

	goPath, err := setupTestFile(configString)

	// Only run the failing part when a specific env variable is set
	if os.Getenv("TEST_CONFIG_VERSION_NOT_SET") == "1" {
		ConfigValidation(tstFilePath)
		return
	}

	// Start the actual test in a different subprocess
	cmd := exec.Command(os.Args[0], "-test.run=TestConfigVersionNotSet")
	cmd.Env = append(os.Environ(), "TEST_CONFIG_VERSION_NOT_SET=1")
	if cmdErr := cmd.Start(); cmdErr != nil {
		t.Fatal(cmdErr)
	}

	// Check that the program exited
	error := cmd.Wait()
	if e, ok := error.(*exec.ExitError); !ok || e.Success() {
		t.Fatalf("Process ran with err %v, want exit status 1", err)
	}

	// Check if the log message is what we expected
	if _, logErr := os.Stat(goPath + "/" + errFile); os.IsNotExist(logErr) {
		t.Fatal(logErr)
	}
	gotBytes, err := ioutil.ReadFile(goPath + "/" + errFile)
	if err != nil {
		t.Fatal(err)
	}
	got := string(gotBytes)
	expected := "Config Version is missing."
	if !strings.Contains(got, expected) {
		t.Fatalf("Unexpected log message. Got %s but should contain %s", got, expected)
	}
	clearTestFile()
	tearTestCase()
}

func TestUseMemoryLimitInConfig(t *testing.T) {
	setupTestCase()
	configString :=
		`Socket:
  UDPAddress: "127.0.0.1:3000"
MemoryLimit: 8
Region: "us-east-2"
Endpoint: "https://xxxx.xxxx.com"
ResourceARN: ""
RoleARN: ""
Concurrency: 8
Version: 1`

	goPath, err := setupTestFile(configString)

	// Only run the failing part when a specific env variable is set
	if os.Getenv("TEST_USE_MEMORYLIMIT_FLAG") == "1" {
		ConfigValidation(tstFilePath)
		return
	}

	// Start the actual test in a different subprocess
	cmd := exec.Command(os.Args[0], "-test.run=TestUseMemoryLimitInConfig")
	cmd.Env = append(os.Environ(), "TEST_USE_MEMORYLIMIT_FLAG=1")
	if cmdErr := cmd.Start(); cmdErr != nil {
		t.Fatal(cmdErr)
	}

	// Check that the program exited
	error := cmd.Wait()
	if e, ok := error.(*exec.ExitError); !ok || e.Success() {
		t.Fatalf("Process ran with err %v, want exit status 1", err)
	}

	// Check if the log message is what we expected
	if _, logErr := os.Stat(goPath + "/" + errFile); os.IsNotExist(logErr) {
		t.Fatal(logErr)
	}
	gotBytes, err := ioutil.ReadFile(goPath + "/" + errFile)
	if err != nil {
		t.Fatal(err)
	}
	got := string(gotBytes)
	expected := "MemoryLimit flag is not supported."
	if !strings.Contains(got, expected) {
		t.Fatalf("Unexpected log message. Got %s but should contain %s", got, expected)
	}
	clearTestFile()
	tearTestCase()
}

func TestConfigValidationForNotSupportFlags(t *testing.T) {
	setupTestCase()
	configString :=
		`Socket:
  BufferSizeKB: 128
Version: 1`

	goPath, err := setupTestFile(configString)

	// Only run the failing part when a specific env variable is set
	if os.Getenv("TEST_NOT_SUPPORT_FLAG") == "1" {
		ConfigValidation(tstFilePath)
		return
	}

	// Start the actual test in a different subprocess
	cmd := exec.Command(os.Args[0], "-test.run=TestConfigValidationForNotSupportFlags")
	cmd.Env = append(os.Environ(), "TEST_NOT_SUPPORT_FLAG=1")
	if cmdErr := cmd.Start(); cmdErr != nil {
		t.Fatal(cmdErr)
	}

	// Check that the program exited
	error := cmd.Wait()
	if e, ok := error.(*exec.ExitError); !ok || e.Success() {
		t.Fatalf("Process ran with err %v, want exit status 1", err)
	}

	// Check if the log message is what we expected
	if _, logErr := os.Stat(goPath + "/" + errFile); os.IsNotExist(logErr) {
		t.Fatal(logErr)
	}
	gotBytes, err := ioutil.ReadFile(goPath + "/" + errFile)
	if err != nil {
		t.Fatal(err)
	}
	got := string(gotBytes)
	expected := "Socket.BufferSizeKB flag is not supported any more."
	if !strings.Contains(got, expected) {
		t.Fatalf("Unexpected log message. Got %s but should contain %s", got, expected)
	}
	clearTestFile()
	tearTestCase()
}

func TestConfigValidationForNeedMigrationFlag(t *testing.T) {
	setupTestCase()
	configString :=
		`Processor:
  Region: ""
Version: 1`

	goPath, err := setupTestFile(configString)

	// Only run the failing part when a specific env variable is set
	if os.Getenv("TEST_NEED_MIGRATION_FLAG") == "1" {
		ConfigValidation(tstFilePath)
		return
	}

	// Start the actual test in a different subprocess
	cmd := exec.Command(os.Args[0], "-test.run=TestConfigValidationForNeedMigrationFlag")
	cmd.Env = append(os.Environ(), "TEST_NEED_MIGRATION_FLAG=1")
	if cmdErr := cmd.Start(); cmdErr != nil {
		t.Fatal(cmdErr)
	}

	// Check that the program exited
	error := cmd.Wait()
	if e, ok := error.(*exec.ExitError); !ok || e.Success() {
		t.Fatalf("Process ran with err %v, want exit status 1", err)
	}

	// Check if the log message is what we expected
	if _, logErr := os.Stat(goPath + "/" + errFile); os.IsNotExist(logErr) {
		t.Fatal(logErr)
	}
	gotBytes, err := ioutil.ReadFile(goPath + "/" + errFile)
	if err != nil {
		t.Fatal(err)
	}
	got := string(gotBytes)
	expected := "Processor.Region flag is not supported. Use X-Ray Daemon Config Migration Script to update the config file."
	if !strings.Contains(got, expected) {
		t.Fatalf("Unexpected log message. Got %s but should contain %s", got, expected)
	}
	clearTestFile()
	tearTestCase()
}

func TestConfigValidationForInvalidFlag(t *testing.T) {
	setupTestCase()
	configString := `ABCDE: true
Version: 1`

	goPath := os.Getenv("PWD")
	if goPath == "" {
		panic("GOPATH not set")
	}
	testFile := goPath + "/test_config.yaml"
	f, err := os.Create(testFile)
	if err != nil {
		panic(err)
	}
	f.WriteString(configString)
	f.Close()

	// Only run the failing part when a specific env variable is set
	if os.Getenv("TEST_INVALID_FLAG") == "1" {
		ConfigValidation(testFile)
		return
	}

	// Start the actual test in a different subprocess
	cmd := exec.Command(os.Args[0], "-test.run=TestConfigValidationForInvalidFlag")
	cmd.Env = append(os.Environ(), "TEST_INVALID_FLAG=1")
	if cmdErr := cmd.Start(); cmdErr != nil {
		t.Fatal(cmdErr)
	}

	// Check that the program exited
	error := cmd.Wait()
	if e, ok := error.(*exec.ExitError); !ok || e.Success() {
		t.Fatalf("Process ran with err %v, want exit status 1", err)
	}

	// Check if the log message is what we expected
	if _, logErr := os.Stat(goPath + "/" + errFile); os.IsNotExist(logErr) {
		t.Fatal(logErr)
	}
	gotBytes, err := ioutil.ReadFile(goPath + "/" + errFile)
	if err != nil {
		t.Fatal(err)
	}
	got := string(gotBytes)
	expected := "ABCDE flag is invalid."
	if !strings.Contains(got, expected) {
		t.Fatalf("Unexpected log message. Got %s but should contain %s", got, expected)
	}
	os.Remove(testFile)
	tearTestCase()
}

func TestValidConfigArray(t *testing.T) {
	validString := []string{"TotalBufferSizeMB", "Concurrency", "Endpoint", "Region", "Socket.UDPAddress", "Logging.LogRotation", "Logging.LogLevel", "Logging.LogPath",
		"LocalMode", "ResourceARN", "RoleARN", "NoVerifySSL", "ProxyAddress", "Version"}
	testString := validConfigArray()
	if len(validString) != len(testString) {
		t.Fatalf("Unexpect test array length. Got %v but should be %v", len(testString), len(validString))
	}
	for i, v := range validString {
		if v != testString[i] {
			t.Fatalf("Unexpect Flag in test array. Got %v but should be %v", testString[i], v)
		}
	}
}

func TestUserConfigArray(t *testing.T) {
	configString :=
		`Socket:
  UDPAddress: "127.0.0.1:3000"
MemoryLimit: 8
Region: "us-east-2"
Endpoint: "https://xxxx.xxxx.com"
ResourceARN: ""
RoleARN: ""
Version: 1`

	setupTestFile(configString)

	validString := []string{"Socket.UDPAddress", "MemoryLimit", "Region", "Endpoint", "ResourceARN", "RoleARN", "Version"}
	testString := userConfigArray(tstFilePath)
	if len(validString) != len(testString) {
		t.Fatalf("Unexpect test array length. Got %v but should be %v", len(testString), len(validString))
	}
	for i, v := range validString {
		if v != testString[i] {
			t.Fatalf("Unexpect Flag in test array. Got %v but should be %v", testString[i], v)
		}
	}
	clearTestFile()
}

func TestErrorAndExitForGivenString(t *testing.T) {
	setupTestCase()
	// Only run the failing part when a specific env variable is set
	if os.Getenv("TEST_STRING_ERROR") == "1" {
		errorAndExit("error occurred", nil)
		return
	}

	// Start the actual test in a different subprocess
	cmd := exec.Command(os.Args[0], "-test.run=TestErrorAndExitForGivenString")
	cmd.Env = append(os.Environ(), "TEST_STRING_ERROR=1")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}

	// Check that the program exited
	error := cmd.Wait()
	if e, ok := error.(*exec.ExitError); !ok || e.Success() {
		t.Fatalf("Process ran with err %v, want exit status 1", e)
	}

	// Check if the log message is what we expected
	goPath := os.Getenv("PWD")
	if goPath == "" {
		panic("GOPATH not set")
	}
	if _, err := os.Stat(goPath + "/" + errFile); os.IsNotExist(err) {
		t.Fatal(err)
	}
	gotBytes, err := ioutil.ReadFile(goPath + "/" + errFile)
	if err != nil {
		t.Fatal(err)
	}
	got := string(gotBytes)
	expected := "error occurred"
	if !strings.Contains(got, expected) {
		t.Fatalf("Unexpected log message. Got %s but should contain %s", got, expected)
	}
	tearTestCase()
}

func TestErrorAndExitForGivenError(t *testing.T) {
	setupTestCase()
	if os.Getenv("TEST_ERROR") == "1" {
		err := errors.New("this is an error")
		errorAndExit("", err)
		return
	}

	// Start the actual test in a different subprocess
	cmd := exec.Command(os.Args[0], "-test.run=TestErrorAndExitForGivenError")
	cmd.Env = append(os.Environ(), "TEST_ERROR=1")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}

	// Check that the program exited
	error := cmd.Wait()
	if e, ok := error.(*exec.ExitError); !ok || e.Success() {
		t.Fatalf("Process ran with err %v, want exit status 1", e)
	}

	// Check if the log message is what we expected
	goPath := os.Getenv("PWD")
	if goPath == "" {
		panic("GOPATH not set")
	}
	if _, err := os.Stat(goPath + "/" + errFile); os.IsNotExist(err) {
		t.Fatal(err)
	}
	gotBytes, err := ioutil.ReadFile(goPath + "/" + errFile)
	if err != nil {
		t.Fatal(err)
	}
	got := string(gotBytes)
	expected := "this is an error"
	if !strings.Contains(got, expected) {
		t.Fatalf("Unexpected log message. Got %s but should contain %s", got, expected)
	}
	tearTestCase()
}
