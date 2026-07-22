#!/usr/bin/env bash
#
# Availability + integrity check for the X-Ray daemon's S3-hosted installers
# and executables. Invoked by the monitor-s3 job in
# .github/workflows/continuous-monitoring.yml.
#
# For each 3.x artifact in the origin bucket it publishes CloudWatch metrics to
# the MonitorDaemon namespace, matching the ECR/Dockerhub jobs' convention
# (failure=rate dimension, 1 on failure / 0 on success), plus an artifact
# dimension so an alarm names the exact file that failed:
#   - DownloadFailureFromS3
#   - SignatureVerificationFailureFromS3
#
# Zip artifacts are gpg-verified and rpm artifacts are rpm-verified. DEB
# artifacts are checked for availability only (see the DEB section for why).
#
# Environment:
#   BUCKET_REGION  S3 region to check (default us-east-2). A region matrix can
#                  set this per-leg without changing this script.
#   DRY_RUN        If set to 1, metrics are printed instead of sent to
#                  CloudWatch, so the script runs locally with no AWS creds.
#
# Usage:
#   BUCKET_REGION=us-east-2 .github/scripts/monitor-s3-assets.sh
#   DRY_RUN=1 BUCKET_REGION=us-east-2 .github/scripts/monitor-s3-assets.sh
set -uo pipefail

BUCKET_REGION="${BUCKET_REGION:-us-east-2}"
DRY_RUN="${DRY_RUN:-0}"

BASE="https://s3.${BUCKET_REGION}.amazonaws.com/aws-xray-assets.${BUCKET_REGION}/xray-daemon"
WORK="$(mktemp -d)"
cd "$WORK" || exit 1

# Publish one metric point. In DRY_RUN mode, print instead of calling AWS so
# the script is runnable locally without credentials.
emit() { # $1=metric-name  $2=value(0|1)  $3=artifact
  if [[ "$DRY_RUN" == "1" ]]; then
    echo "[dry-run] $1{artifact=$3} = $2"
  else
    aws cloudwatch put-metric-data --metric-name "$1" --dimensions failure=rate,artifact="$3" --namespace MonitorDaemon --value "$2" --timestamp "$(date +%s)"
  fi
}

# curl -f exits non-zero on any HTTP error; this tests the real public customer
# download path (objects are public, no S3 read perms needed).
download() { # $1=url  $2=outfile
  curl -fsSL --retry 2 -o "$2" "$1"
}

# ---- Public signing key (needed for zip + rpm verification) ------------------
KEY_OK=0
if download "$BASE/aws-xray.gpg" aws-xray.gpg; then
  gpg --import aws-xray.gpg || KEY_OK=1
else
  KEY_OK=1
fi
emit DownloadFailureFromS3 "$KEY_OK" "aws-xray.gpg"

# ---- Zip artifacts: download + documented gpg --verify flow ------------------
ZIP_ARTIFACTS=(
  "aws-xray-daemon-linux-3.x.zip"
  "aws-xray-daemon-linux-arm64-3.x.zip"
  "aws-xray-daemon-macos-3.x.zip"
  "aws-xray-daemon-windows-process-3.x.zip"
  "aws-xray-daemon-windows-service-3.x.zip"
)
for name in "${ZIP_ARTIFACTS[@]}"; do
  dl=0; sig=0
  if download "$BASE/$name" "$name" && download "$BASE/$name.sig" "$name.sig"; then
    if [[ "$KEY_OK" -eq 0 ]] && gpg --verify "$name.sig" "$name"; then
      sig=0
    else
      sig=1
    fi
  else
    dl=1; sig=1   # can't verify what didn't download
  fi
  emit DownloadFailureFromS3 "$dl" "$name"
  emit SignatureVerificationFailureFromS3 "$sig" "$name"
done

# ---- RPM artifacts: download + rpm signature check ---------------------------
# rpm verification runs inside an Amazon Linux container, not on the
# ubuntu-latest host: modern Ubuntu's rpm uses the Sequoia backend, which
# rejects the daemon's (older RSA) signing key and would make every check
# falsely report SIGNATURES NOT OK. Amazon Linux is also the platform these
# RPMs actually target.
RPM_ARTIFACTS=(
  "aws-xray-daemon-3.x.rpm"
  "aws-xray-daemon-arm64-3.x.rpm"
)
for name in "${RPM_ARTIFACTS[@]}"; do
  dl=0; sig=0
  if download "$BASE/$name" "$name"; then
    if [[ "$KEY_OK" -eq 0 ]] && docker run --rm -v "$WORK":/w -w /w amazonlinux:2023 \
         bash -c "rpm --import aws-xray.gpg && rpm -K '$name' | grep -q 'signatures OK'"; then
      sig=0
    else
      sig=1
    fi
  else
    dl=1; sig=1
  fi
  emit DownloadFailureFromS3 "$dl" "$name"
  emit SignatureVerificationFailureFromS3 "$sig" "$name"
done

# ---- DEB artifacts: download availability only -------------------------------
# TODO: DEB signature verification uses the _gpgorigin member written by
# Tool/src/packaging/sign-packages.sh. Verifying it robustly needs the exact
# member concatenation order and is fragile enough to risk false alarms, so it
# is deferred to its own change.
DEB_ARTIFACTS=(
  "aws-xray-daemon-3.x.deb"
  "aws-xray-daemon-arm64-3.x.deb"
)
for name in "${DEB_ARTIFACTS[@]}"; do
  dl=0
  download "$BASE/$name" "$name" || dl=1
  emit DownloadFailureFromS3 "$dl" "$name"
done
