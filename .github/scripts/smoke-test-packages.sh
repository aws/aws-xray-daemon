#!/usr/bin/env bash
#
# Install-and-run smoke test for the X-Ray daemon's S3-hosted OS packages.
# Invoked by the install-smoke-test workflow. Complements the download +
# signature checks in continuous-monitoring.yml: this proves a downloaded
# package actually installs and the daemon binary starts, the S3-package
# equivalent of the existing container-image startup check.
#
# For each package it publishes a CloudWatch metric to the MonitorDaemon
# namespace, matching the existing convention (failure=rate dimension, 0 on
# success / 1 on failure) plus an artifact dimension:
#   - PackageInstallFailureFromS3
#
# One package is tested per invocation (selected by PACKAGE), so the workflow
# can run them as a matrix across native and emulated architectures.
#
# Environment (required):
#   PACKAGE   one of: rpm-x86_64 | rpm-arm64 | deb-x86_64 | deb-arm64
# Environment (optional):
#   DRY_RUN   if 1, print the metric instead of calling CloudWatch (no creds).
#
# The runner must be able to execute the target architecture (native, or QEMU
# via docker/setup-qemu-action for the arm64 variants).
set -uo pipefail

PACKAGE="${PACKAGE:?set PACKAGE to rpm-x86_64|rpm-arm64|deb-x86_64|deb-arm64}"
DRY_RUN="${DRY_RUN:-0}"

BASE="https://s3.us-east-2.amazonaws.com/aws-xray-assets.us-east-2/xray-daemon"

# Map PACKAGE -> artifact filename, docker platform, and base image.
case "$PACKAGE" in
  rpm-x86_64) FILE="aws-xray-daemon-3.x.rpm";        PLATFORM="linux/amd64"; IMAGE="amazonlinux:2023" ;;
  rpm-arm64)  FILE="aws-xray-daemon-arm64-3.x.rpm";  PLATFORM="linux/arm64"; IMAGE="amazonlinux:2023" ;;
  deb-x86_64) FILE="aws-xray-daemon-3.x.deb";        PLATFORM="linux/amd64"; IMAGE="debian:12" ;;
  deb-arm64)  FILE="aws-xray-daemon-arm64-3.x.deb";  PLATFORM="linux/arm64"; IMAGE="debian:12" ;;
  *) echo "unknown PACKAGE: $PACKAGE" >&2; exit 2 ;;
esac

emit() { # $1=value(0|1)
  if [[ "$DRY_RUN" == "1" ]]; then
    echo "[dry-run] PackageInstallFailureFromS3{artifact=$FILE} = $1"
  else
    aws cloudwatch put-metric-data --metric-name PackageInstallFailureFromS3 \
      --dimensions failure=rate,artifact="$FILE" --namespace MonitorDaemon \
      --value "$1" --timestamp "$(date +%s)"
  fi
}

WORK="$(mktemp -d)"
if ! curl -fsSL --retry 2 -o "$WORK/$FILE" "$BASE/$FILE"; then
  echo "FAIL: could not download $FILE"
  emit 1
  exit 0
fi

# Install the package and run the daemon briefly. Success is defined by the
# daemon printing its startup banner ("Initializing AWS X-Ray daemon") -- the
# same signal a healthy start produces. The process exit code is not used: it
# is piped to `head`, which makes the code unreliable (SIGPIPE), and the daemon
# runs until killed anyway.
#
# rpm: the package's preinstall scriptlet runs `useradd`, absent from the
# minimal image, so shadow-utils is installed first (a real host has it). The
# posttrans scriptlet tries to start the systemd service, which cannot work in
# a plain container; that failure is expected and does not affect whether the
# binary installed, which is what we check.
case "$PACKAGE" in
  rpm-*)
    INSTALL='yum install -y shadow-utils >/dev/null 2>&1; yum install -y "/w/'"$FILE"'" >/dev/null 2>&1 || true' ;;
  deb-*)
    INSTALL='apt-get update -qq >/dev/null 2>&1; dpkg -i "/w/'"$FILE"'" >/dev/null 2>&1 || apt-get install -f -y >/dev/null 2>&1' ;;
esac

echo "Testing $PACKAGE ($FILE) on $IMAGE [$PLATFORM]"
out="$(docker run --rm --platform "$PLATFORM" -v "$WORK":/w "$IMAGE" bash -c "
  $INSTALL
  [ -x /usr/bin/xray ] || { echo 'BINARY_MISSING'; exit 0; }
  timeout 5 /usr/bin/xray -o -n us-west-2 2>&1 | head -5
" 2>/dev/null)"

echo "$out"
if grep -q "Initializing AWS X-Ray daemon" <<< "$out"; then
  echo "RESULT: $PACKAGE installed and started OK"
  emit 0
else
  echo "RESULT: $PACKAGE FAILED (no startup banner; binary missing or would not run)"
  emit 1
fi
