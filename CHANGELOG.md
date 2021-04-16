# Change Log
All notable changes to this project will be documented in this file.

## 3.3.1 (2021-04-13)
- Fix nil pointer dereference when the daemon receives an invalid message [PR #122](https://github.com/aws/aws-xray-daemon/pull/122)

## 3.3.0 (2021-04-13)
- Support for fetching region from ECS metadata [PR #41](https://github.com/aws/aws-xray-daemon/pull/41)
- Building X-Ray Daemon docker image from `scratch` instead of from `amazonlinux` as done previously [PR #44](https://github.com/aws/aws-xray-daemon/pull/44)
- Added license and third party licenses to all the packages [PR #46](https://github.com/aws/aws-xray-daemon/pull/46)
- Set hostname and instance-id when EC2 metadata is blocked [PR #54](https://github.com/aws/aws-xray-daemon/pull/54)
- Prevent leaking customer traces at default log level [PR #61](https://github.com/aws/aws-xray-daemon/pull/61)
- Do not print successfully if there are Unprocessed segments [PR #64](https://github.com/aws/aws-xray-daemon/pull/64)
- Reduce debug logging causing log flooding [PR #65](https://github.com/aws/aws-xray-daemon/pull/65)
- Debug logging print only unprocessed segments detail [PR #66](https://github.com/aws/aws-xray-daemon/pull/66)
- Change linux service type from `simple` to `exec` [PR #98](https://github.com/aws/aws-xray-daemon/pull/98)
- Log traceId for unprocessed segments [PR #106](https://github.com/aws/aws-xray-daemon/pull/106)

## 3.2.0 (2019-11-26)
- Do not fail when cleaning build folder
- Bumping up aws-sdk-go to enable IAM Roles for Service Account support in Kubernetes deployments
- xray.service: explicitly set ConfigurationDirectoryMode to 0755

## 3.1.0 (2019-07-01)
- STS credentials are fetched from STS regional endpoint instead of global endpoint. If the configured region is disabled for the STS token, the credentials are 
fetched from primary regional endpoint in that AWS partition. The primary regional endpoint cannot be disabled for STS token.
- Updated AWS Go dependency to version 1.20.10
- Added debug log on request received for HTTP proxy server
- Added info log for the X-Ray service endpoint used by HTTP Proxy server
- Updated X-Ray service endpoint resolution for HTTP Proxy server

## 3.0.2 (2019-06-19)
- Reconfiguring ring buffer channel size to number of buffers allocated to the daemon instead of fix 250 traces

## 3.0.1 (2019-04-16)
- Removed allocating 64KB size to UDP socket connection. Now, the daemon relies on underlying OS default UDP receiver size for the UDP socket connection
- Updated readme about credential configuration: [PR #21](https://github.com/aws/aws-xray-daemon/pull/21)

## 3.0.0 (2018-08-28)
- The daemon now serves as a proxy to the X-Ray SDK for API calls that are related to sampling rules. The proxy runs default on TCP port 2000 and relays calls to get sampling rules and report sampling statistics to X-Ray. The TCP proxy server address can be configured from command line using `t` flag or by using `cfg.yaml` version `2` file.
- `cfg.yaml` file version is changed to `2` and has an extra attribute. The daemon supports version `1` of cfg.yaml:

```
  Socket:
  # Change the address and port on which the daemon listens for HTTP requests to proxy to AWS X-Ray.
  TCPAddress: "127.0.0.1:2000"
```
- Adds timestamp header `X-Amzn-Xray-Timestamp` to PutTraceSegments API calls made by the daemon
- Adding support for configuring `ProxyAddress` through command line: PR [#10](https://github.com/aws/aws-xray-daemon/pull/10)
- AWS X-Ray Daemon source code supports Go Version 1.8 and later


## 2.1.2 (2018-05-14)
- SystemD service file updates for Debian and Linux binaries: PR [#3](https://github.com/aws/aws-xray-daemon/pull/3)
- Added Travis CI: PR [#7](https://github.com/aws/aws-xray-daemon/pull/7)
- Updated service spec files for debian and linux binaries to wait for network to be available : PR [#6](https://github.com/aws/aws-xray-daemon/pull/6)
- Added more unit tests for `conn` package

## 2.1.1 (2018-04-25)
- This version of the AWS X-Ray daemon fixes an issue
where the daemon overrides the customer provided IAM role ARN configuration with the IAM role assigned to EC2 instance profile if a region wasn’t specified in the configuration. We recommend updating to the latest version at the earliest. The latest version of the daemon is available at https://docs.aws.amazon.com/xray/latest/devguide/xray-daemon.html

## 2.1.0 (2018-03-08)
- Open sourced the X-Ray daemon project
- To not upload telemetry data if no traces are recorded
- The daemon logs error to stderr if it fails to read provided configuration file
