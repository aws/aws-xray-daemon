# Change Log
All notable changes to this project will be documented in this file.

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
