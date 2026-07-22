#!/usr/bin/env bash
#
# Availability + integrity check for the X-Ray daemon's S3-hosted installers
# and executables, across every region the asset bucket is replicated to.
# Invoked by the monitor-s3 job in .github/workflows/continuous-monitoring.yml.
#
# For each 3.x artifact it publishes two CloudWatch metrics to the MonitorDaemon
# namespace, matching the ECR/Dockerhub jobs' convention (failure=rate
# dimension) plus an artifact dimension so an alarm names the exact file:
#   - DownloadFailureFromS3
#   - SignatureVerificationFailureFromS3
#
# The metric VALUE is a roll-up count: the number of regions in which that
# artifact failed (0 = healthy everywhere). This keeps the metric count flat as
# regions are added. When a metric is non-zero the run log names the specific
# failing regions, so an alarm on value>0 tells you WHAT failed and the log
# tells you WHERE.
#
# Zip artifacts are gpg-verified and rpm artifacts are rpm-verified. DEB
# artifacts are checked for availability only (see the DEB section for why).
#
# Environment:
#   MONITOR_REGIONS  Space-separated region override (default: the full
#                    replicated list below). Handy for local testing on a
#                    subset, e.g. MONITOR_REGIONS="us-east-2 us-west-2".
#   DRY_RUN          If 1, metrics are printed instead of sent to CloudWatch,
#                    so the script runs locally with no AWS credentials.
#
# Usage:
#   .github/scripts/monitor-s3-assets.sh
#   DRY_RUN=1 MONITOR_REGIONS="us-east-2 us-west-2" .github/scripts/monitor-s3-assets.sh
set -uo pipefail

DRY_RUN="${DRY_RUN:-0}"

# Regions where aws-xray-assets is replicated, discovered by probing the public
# bucket. me-south-1 is deliberately excluded: it is an opt-in region whose S3
# endpoint does not connect from a standard runner, so including it would
# produce a permanent false failure. It needs separate handling before it can
# be monitored here.
REGIONS=(
  us-east-1 us-east-2 us-west-1 us-west-2 ca-central-1
  eu-west-1 eu-west-2 eu-west-3 eu-central-1 eu-north-1 eu-south-1
  ap-south-1 ap-southeast-1 ap-southeast-2 ap-southeast-3
  ap-northeast-1 ap-northeast-2 ap-northeast-3 ap-east-1
  sa-east-1 af-south-1
)
if [[ -n "${MONITOR_REGIONS:-}" ]]; then
  read -r -a REGIONS <<< "$MONITOR_REGIONS"
fi

KEY="aws-xray.gpg"
ZIP_ARTIFACTS=(
  "aws-xray-daemon-linux-3.x.zip"
  "aws-xray-daemon-linux-arm64-3.x.zip"
  "aws-xray-daemon-macos-3.x.zip"
  "aws-xray-daemon-windows-process-3.x.zip"
  "aws-xray-daemon-windows-service-3.x.zip"
)
RPM_ARTIFACTS=(
  "aws-xray-daemon-3.x.rpm"
  "aws-xray-daemon-arm64-3.x.rpm"
)
DEB_ARTIFACTS=(
  "aws-xray-daemon-3.x.deb"
  "aws-xray-daemon-arm64-3.x.deb"
)

WORK="$(mktemp -d)"
cd "$WORK" || exit 1

# Failures are recorded as one "metric artifact region" line per failure in a
# flat log, rather than associative arrays -- macOS still ships bash 3.2, which
# has no `declare -A`, and keeping this portable preserves local testability.
# Counts and failing-region lists are derived from the log at the end.
FAILLOG="$WORK/failures.log"
: > "$FAILLOG"

record_dl_fail()  { echo "DownloadFailureFromS3 $1 $2" >> "$FAILLOG"; }
record_sig_fail() { echo "SignatureVerificationFailureFromS3 $1 $2" >> "$FAILLOG"; }

base_for() { echo "https://s3.$1.amazonaws.com/aws-xray-assets.$1/xray-daemon"; }

# curl -f exits non-zero on any HTTP error; this tests the real public customer
# download path (objects are public, no S3 read perms needed).
download() { curl -fsSL --retry 2 -o "$2" "$1"; }

# Publish one metric point. In DRY_RUN mode, print instead of calling AWS so the
# script is runnable locally without credentials.
emit() { # $1=metric-name  $2=value(count)  $3=artifact
  if [[ "$DRY_RUN" == "1" ]]; then
    echo "[dry-run] $1{artifact=$3} = $2"
  else
    aws cloudwatch put-metric-data --metric-name "$1" --dimensions failure=rate,artifact="$3" --namespace MonitorDaemon --value "$2" --timestamp "$(date +%s)"
  fi
}

# Import a verification key once, from the origin (us-east-2) bucket. If it is
# unavailable, signatures cannot be verified anywhere and every region's sig is
# counted as failed.
KEY_OK=0
if download "$(base_for us-east-2)/$KEY" verify-key.gpg; then
  gpg --import verify-key.gpg || KEY_OK=1
else
  KEY_OK=1
fi

echo "Checking ${#REGIONS[@]} region(s); verification key import: $([[ $KEY_OK -eq 0 ]] && echo ok || echo FAILED)"

for region in "${REGIONS[@]}"; do
  BASE="$(base_for "$region")"
  rdir="$WORK/$region"; mkdir -p "$rdir"

  # --- signing key availability (per region) ---
  download "$BASE/$KEY" "$rdir/$KEY" || record_dl_fail "$KEY" "$region"

  # --- zips: download artifact + .sig, then gpg --verify ---
  for name in "${ZIP_ARTIFACTS[@]}"; do
    if download "$BASE/$name" "$rdir/$name" && download "$BASE/$name.sig" "$rdir/$name.sig"; then
      if [[ "$KEY_OK" -ne 0 ]] || ! gpg --verify "$rdir/$name.sig" "$rdir/$name" >/dev/null 2>&1; then
        record_sig_fail "$name" "$region"
      fi
    else
      record_dl_fail "$name" "$region"
      record_sig_fail "$name" "$region"   # can't verify what didn't download
    fi
  done

  # --- rpms: download, then verify all present ones in one Amazon Linux
  # container (see note below). One container per region keeps mapping simple. ---
  # rpm verification runs inside amazonlinux:2023, not on the ubuntu-latest
  # host: modern Ubuntu's rpm uses the Sequoia backend, which rejects the
  # daemon's (older RSA) signing key and would falsely report SIGNATURES NOT OK.
  rpm_present=()
  for name in "${RPM_ARTIFACTS[@]}"; do
    if download "$BASE/$name" "$rdir/$name"; then
      rpm_present+=("$name")
    else
      record_dl_fail "$name" "$region"
      record_sig_fail "$name" "$region"
    fi
  done
  if [[ ${#rpm_present[@]} -gt 0 ]]; then
    if [[ "$KEY_OK" -ne 0 ]]; then
      for name in "${rpm_present[@]}"; do record_sig_fail "$name" "$region"; done
    else
      cp verify-key.gpg "$rdir/verify-key.gpg"
      verify_out="$(docker run --rm -v "$rdir":/w -w /w amazonlinux:2023 \
        bash -c 'rpm --import verify-key.gpg >/dev/null 2>&1 || exit 3
for f in "$@"; do
  if rpm -K "$f" | grep -q "signatures OK"; then echo "OK $f"; else echo "BAD $f"; fi
done' _ "${rpm_present[@]}" 2>/dev/null)"
      if [[ $? -eq 3 ]]; then
        for name in "${rpm_present[@]}"; do record_sig_fail "$name" "$region"; done
      else
        for name in "${rpm_present[@]}"; do
          grep -q "OK $name" <<< "$verify_out" || record_sig_fail "$name" "$region"
        done
      fi
    fi
  fi

  # --- debs: download availability only ---
  # TODO: DEB signature verification uses the _gpgorigin member written by
  # Tool/src/packaging/sign-packages.sh. Verifying it robustly needs the exact
  # member concatenation order and is fragile enough to risk false alarms, so it
  # is deferred to its own change.
  for name in "${DEB_ARTIFACTS[@]}"; do
    download "$BASE/$name" "$rdir/$name" || record_dl_fail "$name" "$region"
  done
done

# Emit one roll-up metric per artifact; the value is the number of regions in
# which it failed, and the log names those regions when non-zero.
emit_rollup() { # $1=metric  $2=artifact
  # Lines in the fail log for this metric+artifact; the trailing field is region.
  local matches count regions
  matches="$(grep -E "^$1 $2 " "$FAILLOG" 2>/dev/null)"
  count="$(printf '%s' "$matches" | grep -c . )"
  if [[ "$count" -gt 0 ]]; then
    regions="$(printf '%s\n' "$matches" | awk '{print $NF}' | sort -u | tr '\n' ' ')"
    echo "FAIL: $1 {artifact=$2} in $count region(s): $regions"
  fi
  emit "$1" "$count" "$2"
}

for name in "$KEY" "${ZIP_ARTIFACTS[@]}" "${RPM_ARTIFACTS[@]}" "${DEB_ARTIFACTS[@]}"; do
  emit_rollup DownloadFailureFromS3 "$name"
done
for name in "${ZIP_ARTIFACTS[@]}" "${RPM_ARTIFACTS[@]}"; do
  emit_rollup SignatureVerificationFailureFromS3 "$name"
done
