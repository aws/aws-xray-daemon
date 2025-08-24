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
	"os"
	"strings"
	"testing"

	"github.com/aws/aws-xray-daemon/pkg/util/test"
	"github.com/stretchr/testify/assert"
)

var tstFileName = "test_config.json"
var tstFilePath string

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





// getRegionFromECSMetadata() returns a valid region from an appropriate JSON file
func TestValidECSRegion(t *testing.T) {
	metadataFile :=
		`{
    "Cluster": "default",
    "ContainerInstanceARN": "arn:aws:ecs:us-east-1:012345678910:container-instance/default/1f73d099-b914-411c-a9ff-81633b7741dd",
    "TaskARN": "arn:aws:ecs:us-east-1:012345678910:task/default/2b88376d-aba3-4950-9ddf-bcb0f388a40c",
    "TaskDefinitionFamily": "console-sample-app-static",
    "TaskDefinitionRevision": "1",
    "ContainerID": "aec2557997f4eed9b280c2efd7afccdcedfda4ac399f7480cae870cfc7e163fd",
    "ContainerName": "simple-app",
    "DockerContainerName": "/ecs-console-sample-app-static-1-simple-app-e4e8e495e8baa5de1a00",
    "ImageID": "sha256:2ae34abc2ed0a22e280d17e13f9c01aaf725688b09b7a1525d1a2750e2c0d1de",
    "ImageName": "httpd:2.4",
    "PortMappings": [
        {
            "ContainerPort": 80,
            "HostPort": 80,
            "BindIp": "0.0.0.0",
            "Protocol": "tcp"
        }
    ],
    "Networks": [
        {
            "NetworkMode": "bridge",
            "IPv4Addresses": [
                "192.0.2.0"
            ]
        }
    ],
    "MetadataFileStatus": "READY",
    "AvailabilityZone": "us-east-1b",
    "HostPrivateIPv4Address": "192.0.2.0",
    "HostPublicIPv4Address": "203.0.113.0"
}`
	setupTestFile(metadataFile)
	env := stashEnv()
	defer popEnv(env)
	os.Setenv("ECS_ENABLE_CONTAINER_METADATA", "true")
	os.Setenv("ECS_CONTAINER_METADATA_FILE", tstFilePath)
	testString := getRegionFromECSMetadata()

	assert.EqualValues(t, "us-east-1", testString)
	clearTestFile()
	os.Clearenv()
}

// getRegionFromECSMetadata() returns an empty string if ECS metadata related env is not set
func TestNoECSMetadata(t *testing.T){
	env := stashEnv()
	defer popEnv(env)
	testString := getRegionFromECSMetadata()

	assert.EqualValues(t, "", testString)
}
// getRegionFromECSMetadata() throws an error and returns an empty string when ECS metadata file cannot be parsed as valid JSON
func TestInvalidECSMetadata(t *testing.T){
	metadataFile := "][foobar})("
	setupTestFile(metadataFile)
	env := stashEnv()
	defer popEnv(env)
	os.Setenv("ECS_ENABLE_CONTAINER_METADATA", "true")
	os.Setenv("ECS_CONTAINER_METADATA_FILE", tstFilePath)
	log := test.LogSetup()

	testString := getRegionFromECSMetadata()

	assert.EqualValues(t, "", testString)
	assert.True(t, strings.Contains(log.Logs[0], "Unable to read"))

	clearTestFile()
}

// getRegionFromECSMetadata() throws an error and returns an empty string when ECS metadata file cannot be opened
func TestMissingECSMetadataFile(t *testing.T){
	metadataFile := "foobar"
	setupTestFile(metadataFile)
	env := stashEnv()
	defer popEnv(env)
	clearTestFile()

	os.Setenv("ECS_ENABLE_CONTAINER_METADATA", "true")
	os.Setenv("ECS_CONTAINER_METADATA_FILE", metadataFile)
	log := test.LogSetup()

	testString := getRegionFromECSMetadata()

	assert.EqualValues(t, "", testString)
	assert.True(t, strings.Contains(log.Logs[0], "Unable to open"))
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

func TestGetPartition1(t *testing.T) {
	r := "us-east-1"
	p := getPartition(r)
	assert.Equal(t, PartitionAWS, p)
}

func TestGetPartition2(t *testing.T) {
	r := "cn-north-1"
	p := getPartition(r)
	assert.Equal(t, PartitionAWSCN, p)
}

func TestGetPartition3(t *testing.T) {
	r := "us-gov-east-1"
	p := getPartition(r)
	assert.Equal(t, PartitionAWSUSGov, p)
}

func TestGetPartition4(t *testing.T) { // if a region is not present in the array
	r := "XYZ"
	p := getPartition(r)
	assert.Equal(t, "", p)
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
