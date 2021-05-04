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
    s3_bucket_name="aws-xray-assets.staging"
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
	"upload_ready_binaries/${package_name}-linux-amd64-${release_version}.rpm"
  "upload_ready_binaries/${package_name}-linux-amd64-${release_version}.zip"
  "upload_ready_binaries/${package_name}-linux-amd64-${release_version}.zip.sig"
  "upload_ready_binaries/${package_name}-linux-amd64-${major_version}.deb"
  "upload_ready_binaries/${package_name}-linux-amd64-${major_version}.rpm"
  "upload_ready_binaries/${package_name}-linux-amd64-${major_version}.zip"
  "upload_ready_binaries/${package_name}-linux-amd64-${major_version}.zip.sig"
  "upload_ready_binaries/${package_name}-linux-arm64-${release_version}.deb"
  "upload_ready_binaries/${package_name}-linux-arm64-${release_version}.rpm"
  "upload_ready_binaries/${package_name}-linux-arm64-${release_version}.zip"
  "upload_ready_binaries/${package_name}-linux-arm64-${release_version}.zip.sig"
  "upload_ready_binaries/${package_name}-linux-arm64-${major_version}.deb"
  "upload_ready_binaries/${package_name}-linux-arm64-${major_version}.rpm"
  "upload_ready_binaries/${package_name}-linux-arm64-${major_version}.zip"
  "upload_ready_binaries/${package_name}-linux-arm64-${major_version}.zip.sig"
  "upload_ready_binaries/${package_name}-macos-${major_version}.zip"
  "upload_ready_binaries/${package_name}-macos-${major_version}.zip.sig"
  "upload_ready_binaries/${package_name}-macos-amd64-${release_version}.zip"
  "upload_ready_binaries/${package_name}-macos-amd64-${release_version}.zip.sig"
  "upload_ready_binaries/${package_name}-macos-amd64-${major_version}.zip"
  "upload_ready_binaries/${package_name}-macos-amd64-${major_version}.zip.sig"
  "upload_ready_binaries/${package_name}-macos-arm64-${release_version}.zip"
  "upload_ready_binaries/${package_name}-macos-arm64-${release_version}.zip.sig"
  "upload_ready_binaries/${package_name}-macos-arm64-${major_version}.zip"
  "upload_ready_binaries/${package_name}-macos-arm64-${major_version}.zip.sig"
  "upload_ready_binaries/${package_name}-windows-amd64-${release_version}.zip"
  "upload_ready_binaries/${package_name}-windows-amd64-${release_version}.zip.sig"
  "upload_ready_binaries/${package_name}-windows-amd64-${major_version}.zip"
  "upload_ready_binaries/${package_name}-windows-amd64-${major_version}.zip.sig"
  "upload_ready_binaries/${package_name}-windows-amd64-service-${release_version}.zip"
  "upload_ready_binaries/${package_name}-windows-amd64-service-${release_version}.zip.sig"
  "upload_ready_binaries/${package_name}-windows-amd64-service-${major_version}.zip"
  "upload_ready_binaries/${package_name}-windows-amd64-service-${major_version}.zip.sig"
  "upload_ready_binaries/${package_name}-windows-process-${major_version}.zip"
  "upload_ready_binaries/${package_name}-windows-process-${major_version}.zip.sig"
  "upload_ready_binaries/${package_name}-windows-service-${major_version}.zip"
  "upload_ready_binaries/${package_name}-windows-service-${major_version}.zip.sig"
)

# check required files are available
for i in "${required_files[@]}"
do
	if [ ! -f "${i}" ]; then
		echo "${i} does not exist"
		exit 1
	fi
done

# upload daemon binaries to s3
for i in "${required_files[@]}"
do
	aws s3api put-object --bucket "${s3_bucket_name}" --key "${release_version}/${i}" --body "${i}"
done
