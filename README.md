[![Build Status](https://travis-ci.org/aws/aws-xray-daemon.svg?branch=master)](https://travis-ci.org/aws/aws-xray-daemon)
[![Go Report Card](https://goreportcard.com/badge/github.com/aws/aws-xray-daemon)](https://goreportcard.com/report/github.com/aws/aws-xray-daemon)

# AWS X-Ray Daemon  

The AWS X-Ray daemon is a software application that listens for traffic on UDP port 2000, gathers raw segment data, and relays it to the AWS X-Ray API.   
The daemon works in conjunction with the AWS X-Ray SDKs and must be running so that data sent by the SDKs can reach the X-Ray service. For more information,
 see [AWS X-Ray Daemon](https://docs.aws.amazon.com/xray/latest/devguide/xray-daemon.html).

## Getting Help  

Use the following community resources for getting help with the AWS X-Ray Daemon. We use the GitHub issues for tracking bugs and feature requests.  

* Ask a question in the [AWS X-Ray Forum](https://forums.aws.amazon.com/forum.jspa?forumID=241&start=0).  
* Open a support ticket with [AWS Support](http://docs.aws.amazon.com/awssupport/latest/user/getting-started.html).  
* If you think you may have found a bug, open an [issue](https://github.com/aws/aws-xray-daemon/issues/new).  
* For contributing guidelines refer [CONTRIBUTING.md](https://github.com/aws/aws-xray-daemon/blob/master/CONTRIBUTING.md).

## Sending Segment Documents

The X-Ray SDK sends segment documents to the daemon to avoid making calls to AWS directly. You can send the segment/subsegment in JSON over UDP port 2000
to the X-Ray daemon, prepended by the daemon header : `{"format": "json", "version": 1}\n`

```
{"format": "json", "version": 1}\n{<serialized segment data>}
```  
For more details refer : [Link](https://docs.aws.amazon.com/xray/latest/devguide/xray-api-sendingdata.html)  

## Installing  

The AWS X-Ray Daemon is compatible with Go 1.8 and later.

Install the daemon using the following command:  

```  
go get -u github.com/aws/aws-xray-daemon/...  
```  

## Credential Configuration

The AWS X-Ray Daemon follows default credential resolution for the [aws-sdk-go](https://docs.aws.amazon.com/sdk-for-go/api/index.html#hdr-Configuring_Credentials).

Follow the [guidelines](https://docs.aws.amazon.com/sdk-for-go/v1/developer-guide/configuring-sdk.html) for the credential configuration.

## Daemon Usage (command line args)  

Usage: xray [options]   

| | | Description |
| --- | --- | --- |
| -a | --resource-arn | Amazon Resource Name (ARN) of the AWS resource running the daemon. |
| -o | --local-mode | Don't check for EC2 instance metadata. |
| -m | --buffer-memory | Change the amount of memory in MB that buffers can use (minimum 3). |
| -n | --region | Send segments to X-Ray service in a specific region. |
| -b | --bind | Overrides default UDP address (127.0.0.1:2000). |
| -t | --bind-tcp | Overrides default TCP address (127.0.0.1:2000). |
| -r | --role-arn | Assume the specified IAM role to upload segments to a different account. |
| -c | --config | Load a configuration file from the specified path. |
| -f | --log-file | Output logs to the specified file path. |
| -l | --log-level | Log level, from most verbose to least: dev, debug, info, warn, error, prod (default). |
| -p | --proxy-address | Proxy address through which to upload segments. |
| -v | --version | Show AWS X-Ray daemon version. |
| -h | --help | Show this screen |

## Build  

`make build` would build binaries and .zip files in `/build` folder for Linux, MacOS, and Windows platforms.    

### Linux  

`make build-linux` would build binaries and .zip files in `/build` folder for the Linux platform.  

### MAC  

`make build-mac` would build binaries and .zip files in `/build` folder for the MacOS platform.  

### Windows  

`make build-windows` would build binaries and .zip files in `/build` folder for the Windows platform. 

## Build for ARM achitecture
Currently, the `make build` script builds artifacts for AMD architecture. You can build the X-Ray Daemon for ARM by using the `go build` command and setting the `GOARCH` to `arm64`. To build the daemon binary on a linux ARM machine, you can use the following command:
```
GOOS=linux GOARCH=arm64 go build -ldflags "-s -w" -o xray cmd/tracing/daemon.go cmd/tracing/tracing.go
```
As of Aug 31, 2020, windows and darwin builds for ARM64 are not supported by `go build`.

## Pulling X-Ray Daemon image from ECR Public Gallery
Before pulling an image you should authenticate your docker client to the Amazon ECR public registry. For registry authentication options follow this [link](https://docs.aws.amazon.com/AmazonECR/latest/public/public-registries.html#public-registry-auth)

Run below command to authenticate to public ECR registry using `get-login-password` (AWS CLI)

``
aws ecr-public get-login-password --region us-east-1 | docker login --username AWS --password-stdin public.ecr.aws
``

####  Pull alpha tag from Public ECR Gallery
``
docker pull public.ecr.aws/xray/aws-xray-daemon:alpha
``

####  Pull released version tag from Public ECR Gallery
``
docker pull public.ecr.aws/xray/aws-xray-daemon:3.2.0
``

NOTE: We are not recommending to use daemon image with alpha tag in production environment. For production environment customer should pull in an image with released tag. 

## X-Ray Daemon Performance Report

**EC2 Instance Type:** T2.Micro [1 vCPU, 1 GB Memory]

**Collection time:** 10 minutes per TPS (TPS = Number of segments sent to daemon in 1 second)

**Daemon version tested:** 3.3.6

| **TPS** | **Avg CPU Usage (%)** | **Avg Memory Usage (MB)** |
|---------|-----------------------|---------------------------|
| 0       | 0                     | 17.07                     |
| 100     | 0.9                   | 28.5                      |
| 200     | 1.87                  | 29.3                      |
| 400     | 3.76                  | 29.1                      |
| 1000    | 9.36                  | 29.5                      |
| 2000    | 18.9                  | 29.7                      |
| 4000    | 38.3                  | 29.5                      |


## Testing  

`make test` will run unit tests for the X-Ray daemon.  

## License

This library is licensed under the Apache 2.0 License.
