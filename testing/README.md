## Testing X-Ray Daemon

This directory contains the resources and logic for testing the functionality of X-Ray Daemon for various test cases. The testcases leverage Terraform with variable configurations to launch X-Ray Daemon on AWS EC2 instances and send a trace through it. 

Current features of this testing suite are:
1. Testing the `.zip`, `.deb`, and `.rpm` binaries for x86_64 architecture of Linux platform.
2. Testing the `.zip` binary for x86_64 architecture of Linux platform in China and US Gov AWS partitions.
