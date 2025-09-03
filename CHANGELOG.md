# Change Log
All notable changes to this project will be documented in this file.

## 3.6.0 (2025-09-03)
- Add daemon verification steps to continuous-build workflow [PR #248](https://github.com/aws/aws-xray-daemon/pull/248)
- Aws SDK Go v1 -> v2 migration [PR #247](https://github.com/aws/aws-xray-daemon/pull/247)
- Fix UserAgent by modifying the current value rather than adding a separate attribute [PR #253](https://github.com/aws/aws-xray-daemon/pull/253)

## 3.5.0 (2025-08-28)
- Revert migrate AWS SDK for Go from v1 to v2 [PR #245](https://github.com/aws/aws-xray-daemon/pull/245)

## 3.4.0 (2025-08-27)
- Migrate AWS SDK for Go from v1 to v2 [PR #241](https://github.com/aws/aws-xray-daemon/pull/241)

## 3.3.15 (2025-06-25)
- Bump Go version to v1.24.2 [PR #237](https://github.com/aws/aws-xray-daemon/pull/237)
- Bump golang.org/x/net from 0.33.0 to 0.38.0 (#235) [PR #235](https://github.com/aws/aws-xray-daemon/pull/235)

## 3.3.14 (2025-02-12)
- Bump Go version to v1.23.6 [PR #231](https://github.com/aws/aws-xray-daemon/pull/231)
- Bump golang.org/x/net from 0.23.0 to 0.33.0 [PR #230](https://github.com/aws/aws-xray-daemon/pull/230)

## 3.3.13 (2024-07-24)
- Bump Go version to v1.22.4 [PR #222](https://github.com/aws/aws-xray-daemon/pull/222)

## 3.3.12 (2024-05-01)
- Bump golang.org/x/net from 0.17.0 to 0.23.0 [PR #219](https://github.com/aws/aws-xray-daemon/pull/219)

## 3.3.11 (2024-03-27)
- Use latest tag for amazonlinux for Docker image [PR #217](https://github.com/aws/aws-xray-daemon/pull/217)
- Add http2 timouts to close bad TCP connection [PR #216](https://github.com/aws/aws-xray-daemon/pull/216)

## 3.3.10 (2023-12-20)
- Bump Go version to v1.21.5 [PR #212](https://github.com/aws/aws-xray-daemon/pull/212)

## 3.3.9 (2023-10-31)
- Bump golang.org/x/net to v0.17.0 to fix CVE-2023-44487 [PR #208](https://github.com/aws/aws-xray-daemon/pull/208)
- Bump Go version to v1.21.3 [PR #209](https://github.com/aws/aws-xray-daemon/pull/209)

## 3.3.8 (2023-09-08)
- Bump Go version to v1.21.1 [PR #205](https://github.com/aws/aws-xray-daemon/pull/205)
- Bump golang.org/x/net to v0.15.0 to fix CVE-2023-3978 [PR #205](https://github.com/aws/aws-xray-daemon/pull/205)
- Bump aws-sdk-go to v1.44.298 for SSO token provider support for sso-session in AWS shared config [PR #206](https://github.com/aws/aws-xray-daemon/pull/206)

## 3.3.7 (2023-04-24)
- Bump golang.org/x/net to v0.7.0 to fix CVE 2022-41725 [PR #193](https://github.com/aws/aws-xray-daemon/pull/193)
- Bump Go version to 1.20.3 [PR #196](https://github.com/aws/aws-xray-daemon/pull/196)

## 3.3.6 (2023-02-01)
- User-agent redesign - add additional information to user-agent [PR #188](https://github.com/aws/aws-xray-daemon/pull/188)
- Remove custom backoff logic for sending segments [PR #186](https://github.com/aws/aws-xray-daemon/pull/186)

## 3.3.5 (2022-09-22)
- Fix CVE-2022-27664 [PR #180](https://github.com/aws/aws-xray-daemon/pull/180)

## 3.3.4 (2022-08-31)
- Upgrade aws-sdk-go to latest version to get SSO credential support [PR #168](https://github.com/aws/aws-xray-daemon/pull/168)
- Fix CVE issues by bumping GO version to 1.18 [PR #173](https://github.com/aws/aws-xray-daemon/pull/173)

## 3.3.3 (2021-07-21)
- Add logging for ignored errors [PR #138](https://github.com/aws/aws-xray-daemon/pull/138)
- Upgrade minimum golang version for compiling to 1.16.6 [PR #148](https://github.com/aws/aws-xray-daemon/pull/148)
- Upgrade golang `net` and `sys` libraries [PR #150](https://github.com/aws/aws-xray-daemon/pull/150)

## 3.3.2 (2021-04-16)
- Fix Daemon startup log missing, set default log level as Info [PR #129](https://github.com/aws/aws-xray-daemon/pull/129)
- Rollback Dockerhub image base to AmazonLinux [PR #130](https://github.com/aws/aws-xray-daemon/pull/130)

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
