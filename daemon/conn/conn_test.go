// Copyright 2018-2018 Amazon.com, Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may not use this file except in compliance with the License. A copy of the License is located at
//
//     http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package conn

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/aws/aws-xray-daemon/daemon/util/test"

	"github.com/stretchr/testify/mock"

	"github.com/aws/aws-xray-daemon/daemon/cfg"

	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/stretchr/testify/assert"
)

var ec2Region = "us-east-1"

type mockConn struct {
	mock.Mock
	sn *session.Session
}

func (c *mockConn) getEC2Region(s *session.Session) (string, error) {
	args := c.Called(nil)
	errorStr := args.String(0)
	var err error
	if errorStr != "" {
		err = errors.New(errorStr)
		return "", err
	}
	return ec2Region, nil
}

func (c *mockConn) newAWSSession(roleArn string) *session.Session {
	return c.sn
}

// fetch region value from ec2 meta data service
func TestEC2Session(t *testing.T) {
	m := new(mockConn)
	log := test.LogSetup()
	m.On("getEC2Region", nil).Return("").Once()
	var expectedSession *session.Session
	roleARN := ""
	expectedSession, _ = session.NewSession()
	m.sn = expectedSession
	cfg, s := GetAWSConfigSession(m, cfg.DefaultConfig(), roleARN, "", false)
	assert.Equal(t, s, expectedSession, "Expect the session object is not overridden")
	assert.Equal(t, *cfg.Region, ec2Region, "Region value fetched from ec2-metadata service")
	assert.True(t, strings.Contains(log.Logs[0], fmt.Sprintf("Fetch region %v from ec2 metadata", ec2Region)))
}

// fetch region value from environment variable
func TestRegionEnv(t *testing.T) {
	log := test.LogSetup()
	region := "us-west-2"
	env := stashEnv()
	defer popEnv(env)
	os.Setenv("AWS_REGION", region)

	var m = &mockConn{}
	var expectedSession *session.Session
	roleARN := ""
	expectedSession, _ = session.NewSession()
	m.sn = expectedSession
	cfg, s := GetAWSConfigSession(m, cfg.DefaultConfig(), roleARN, "", true)
	assert.Equal(t, s, expectedSession, "Expect the session object is not overridden")
	assert.Equal(t, *cfg.Region, region, "Region value fetched from environment")
	assert.True(t, strings.Contains(log.Logs[0], fmt.Sprintf("Fetch region %v from environment variables", region)))
}

// Get region from the command line fo config file
func TestRegionArgument(t *testing.T) {
	log := test.LogSetup()
	region := "ap-northeast-1"
	var m = &mockConn{}
	var expectedSession *session.Session
	roleARN := ""
	expectedSession, _ = session.NewSession()
	m.sn = expectedSession
	cfg, s := GetAWSConfigSession(m, cfg.DefaultConfig(), roleARN, region, true)
	assert.Equal(t, s, expectedSession, "Expect the session object is not overridden")
	assert.Equal(t, *cfg.Region, region, "Region value fetched from the environment")
	assert.True(t, strings.Contains(log.Logs[0], fmt.Sprintf("Fetch region %v from commandline argument", region)))
}

// exit function if no region value found
func TestNoRegion(t *testing.T) {
	region := ""
	envFlag := "NO_REGION"
	var m = &mockConn{}
	var expectedSession *session.Session
	roleARN := ""
	expectedSession, _ = session.NewSession()
	m.sn = expectedSession

	if os.Getenv(envFlag) == "1" {
		GetAWSConfigSession(m, cfg.DefaultConfig(), roleARN, region, true) // exits because no region found
		return
	}

	// Start the actual test in a different subprocess
	cmd := exec.Command(os.Args[0], "-test.run=TestNoRegion")
	cmd.Env = append(os.Environ(), envFlag+"=1")
	if cmdErr := cmd.Start(); cmdErr != nil {
		t.Fatal(cmdErr)
	}

	// Check that the program exited
	error := cmd.Wait()
	if e, ok := error.(*exec.ExitError); !ok || e.Success() {
		t.Fatalf("Process ran with err %v, want exit status 1", error)
	}
}

// getEC2Region() returns nil region and error, resulting in exiting the process
func TestErrEC2(t *testing.T) {
	m := new(mockConn)
	m.On("getEC2Region", nil).Return("Error").Once()
	var expectedSession *session.Session
	roleARN := ""
	expectedSession, _ = session.NewSession()
	m.sn = expectedSession
	envFlag := "NO_REGION"
	if os.Getenv(envFlag) == "1" {
		GetAWSConfigSession(m, cfg.DefaultConfig(), roleARN, "", false)
		return
	}

	// Start the actual test in a different subprocess
	cmd := exec.Command(os.Args[0], "-test.run=TestErrEC2")
	cmd.Env = append(os.Environ(), envFlag+"=1")
	if cmdErr := cmd.Start(); cmdErr != nil {
		t.Fatal(cmdErr)
	}

	// Check that the program exited
	error := cmd.Wait()
	if e, ok := error.(*exec.ExitError); !ok || e.Success() {
		t.Fatalf("Process ran with err %v, want exit status 1", error)
	}
}

func TestLoadEnvConfigCreds(t *testing.T) {
	env := stashEnv()
	defer popEnv(env)

	cases := struct {
		Env map[string]string
		Val credentials.Value
	}{
		Env: map[string]string{
			"AWS_ACCESS_KEY":    "AKID",
			"AWS_SECRET_KEY":    "SECRET",
			"AWS_SESSION_TOKEN": "TOKEN",
		},
		Val: credentials.Value{
			AccessKeyID: "AKID", SecretAccessKey: "SECRET", SessionToken: "TOKEN",
			ProviderName: "EnvConfigCredentials",
		},
	}

	for k, v := range cases.Env {
		os.Setenv(k, v)
	}
	c := &Conn{}
	cfg := c.newAWSSession("")
	value, err := cfg.Config.Credentials.Get()

	assert.Nil(t, err, "Expect no error")
	assert.Equal(t, cases.Val, value, "Expect the credentials value to match")

	cfgA := c.newAWSSession("ROLEARN")
	valueA, _ := cfgA.Config.Credentials.Get()

	assert.Equal(t, "", valueA.AccessKeyID, "Expect the value to be empty")
	assert.Equal(t, "", valueA.SecretAccessKey, "Expect the value to be empty")
	assert.Equal(t, "", valueA.SessionToken, "Expect the value to be empty")
	assert.Equal(t, "", valueA.ProviderName, "Expect the value to be empty")
}

func TestGetProxyUrlProxyAddressNotValid(t *testing.T) {
	errorAddress := [3]string{"http://[%10::1]", "http://%41:8080/", "http://a b.com/"}
	for _, address := range errorAddress {
		// Only run the failing part when a specific env variable is set
		if os.Getenv("Test_PROXY_URL") == "1" {
			getProxyURL(address)
			return
		}
		// Start the actual test in a different subprocess
		cmd := exec.Command(os.Args[0], "-test.run=TestGetProxyUrlProxyAddressNotValid")
		cmd.Env = append(os.Environ(), "Test_PROXY_URL=1")
		if err := cmd.Start(); err != nil {
			t.Fatal(err)
		}
		// Check that the program exited
		err := cmd.Wait()
		if e, ok := err.(*exec.ExitError); !ok || e.Success() {
			t.Fatalf("Process ran with err %v, want exit status 1", err)
		}
	}
}

func TestGetProxyAddressFromEnvVariable(t *testing.T) {
	env := stashEnv()
	defer popEnv(env)
	os.Setenv("HTTPS_PROXY", "https://127.0.0.1:8888")

	assert.Equal(t, os.Getenv("HTTPS_PROXY"), getProxyAddress(""), "Expect function return value should be same with Environment value")
}

func TestGetProxyAddressFromConfigFile(t *testing.T) {
	env := stashEnv()
	defer popEnv(env)

	assert.Equal(t, "https://127.0.0.1:8888", getProxyAddress("https://127.0.0.1:8888"), "Expect function return value should be same with input value")
}

func TestGetProxyAddressWhenNotExist(t *testing.T) {
	env := stashEnv()
	defer popEnv(env)

	assert.Equal(t, "", getProxyAddress(""), "Expect function return value to be empty")
}

func TestGetProxyAddressPriority(t *testing.T) {
	env := stashEnv()
	defer popEnv(env)
	os.Setenv("HTTPS_PROXY", "https://127.0.0.1:8888")

	assert.Equal(t, "https://127.0.0.1:9999", getProxyAddress("https://127.0.0.1:9999"), "Expect function return value to be same with input")
}

func stashEnv() []string {
	env := os.Environ()
	os.Clearenv()

	return env
}

func popEnv(env []string) {
	os.Clearenv()

	for _, e := range env {
		p := strings.SplitN(e, "=", 2)
		os.Setenv(p[0], p[1])
	}
}
