# Change Log
All notable changes to this project will be documented in this file.

## 3.0.0 (2018-08-28)
- The daemon now serves as a proxy to the X-Ray SDK for API calls that are related to sampling rules. The proxy runs on TCP port 2000 and relays calls to get sampling rules and report sampling statistics to X-Ray. The TCP proxy server address can be configured from command line using `t` flag or by using `cfg.yaml` version `2` file.
- `cfg.yaml` file version is changed to `2` and has an extra attribute. The daemon supports version `1` of cfg.yaml:

```
  Socket:
  # Change the address and port on which the daemon listens for HTTP requests to proxy to AWS X-Ray.
  TCPAddress: "127.0.0.1:2000"
```
- Adds timestamp header `X-Amzn-Xray-Timestamp` to PutTraceSegments API calls made by the daemon
- Adding support for configuring `ProxyAddress` through command line: PR [#10](https://github.com/aws/aws-xray-daemon/pull/10)

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
