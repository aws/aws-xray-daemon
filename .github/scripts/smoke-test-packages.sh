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
# Bounded timeouts so a stalled S3 connection can't hold the hourly run open
# (the job also carries a timeout-minutes guard). --max-time is per attempt;
# --retry can add up to one more.
if ! curl -fsSL --connect-timeout 10 --max-time 120 --retry 2 -o "$WORK/$FILE" "$BASE/$FILE"; then
  echo "FAIL: could not download $FILE"
  emit 1
  exit 0
fi

# Install the package, then confirm it is registered and the daemon starts.
# The check requires, in order: the package operation exits 0; the package
# manager reports it installed (rpm -q / dpkg-query); and the daemon prints its
# startup banner. Any of these missing is a failure. The container prints
# INSTALL_OK / REGISTERED / then the daemon output, and finally CHECK_DONE so a
# container that never ran (Docker/image-pull failure) can't be read as healthy.
#
# The packages' maintainer scripts try to start the systemd service, which
# cannot work in a plain container. Rather than swallow all install errors
# (which would hide a real install failure), the service start is neutralized
# explicitly so the transaction itself can be required to succeed:
#   rpm: stub systemctl to a no-op (also install shadow-utils, which provides
#        the useradd the preinstall scriptlet needs on a minimal image).
#   deb: install a policy-rc.d that denies service starts.
# The installed package name is "xray" for both rpm and deb.
case "$PACKAGE" in
  rpm-*)
    INSTALL='yum install -y shadow-utils >/dev/null 2>&1
      printf "#!/bin/sh\nexit 0\n" > /usr/bin/systemctl; chmod +x /usr/bin/systemctl
      yum install -y "/w/'"$FILE"'" >/dev/null 2>&1 || exit 10
      rpm -q xray >/dev/null 2>&1 || exit 11' ;;
  deb-*)
    INSTALL='apt-get update -qq >/dev/null 2>&1
      printf "#!/bin/sh\nexit 101\n" > /usr/sbin/policy-rc.d; chmod +x /usr/sbin/policy-rc.d
      { dpkg -i "/w/'"$FILE"'" >/dev/null 2>&1 || apt-get install -f -y >/dev/null 2>&1; } || exit 10
      dpkg -s xray 2>/dev/null | grep -q "^Status: install ok installed" || exit 11' ;;
esac

echo "Testing $PACKAGE ($FILE) on $IMAGE [$PLATFORM]"
out="$(docker run --rm --platform "$PLATFORM" -v "$WORK":/w "$IMAGE" bash -c "
  $INSTALL
  echo INSTALL_REGISTERED
  [ -x /usr/bin/xray ] || { echo 'BINARY_MISSING'; exit 12; }
  timeout 5 /usr/bin/xray -o -n us-west-2 2>&1 | head -5
  echo CHECK_DONE
" 2>/dev/null)"
docker_rc=$?

echo "$out"
if [[ "$docker_rc" -ne 0 && "$docker_rc" -ne 124 ]] || ! grep -q "INSTALL_REGISTERED" <<< "$out"; then
  echo "RESULT: $PACKAGE FAILED (install did not succeed / package not registered / verifier could not run; docker rc=$docker_rc)"
  emit 1
elif grep -q "Initializing AWS X-Ray daemon" <<< "$out"; then
  echo "RESULT: $PACKAGE installed, registered, and started OK"
  emit 0
else
  echo "RESULT: $PACKAGE FAILED (installed but daemon did not print startup banner)"
  emit 1
fi
