# Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License").
# You may not use this file except in compliance with the License.
# A copy of the License is located at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# or in the "license" file accompanying this file. This file is distributed
# on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
# express or implied. See the License for the specific language governing
# permissions and limitations under the License.


#! /bin/bash

set -e

##########################################
# This script is used in X-Ray daemon CD workflow
# to publish X-Ray daemon binaries to s3
#
# Env vars
# 1. s3_bucket_name
# 2. release_version
# 3. major_version
##########################################

# check environment vars
if [ -z ${s3_bucket_name} ]; then
    s3_bucket_name="aws-xray-daemon-assets.staging"
	echo "use default s3_bucket_name: ${s3_bucket_name}"
else
	echo "load s3_bucket_name from env var: ${s3_bucket_name}"
fi

# define vars
package_name="aws-xray-daemon"

# check if all the required files are there
declare -a required_files=(
	"upload_ready_binaries/${package_name}-${major_version}.deb"
	"upload_ready_binaries/${package_name}-${major_version}.rpm"
	"upload_ready_binaries/${package_name}-arm64-${major_version}.deb"
	"upload_ready_binaries/${package_name}-arm64-${major_version}.rpm"
	"upload_ready_binaries/${package_name}-linux-${major_version}.zip"
	"upload_ready_binaries/${package_name}-linux-${major_version}.zip.sig"
	"upload_ready_binaries/${package_name}-linux-amd64-${release_version}.deb"
)

for i in "${required_files[@]}"
do
	if [ ! -f "${i}" ]; then
		echo "${i} does not exist"
		exit 1
	fi
done

# upload packages to s3
for i in "${required_files[@]}"
do
	aws s3api put-object --bucket "${s3_bucket_name}" --key "${release_version}/${i}" --body "${i}"
done