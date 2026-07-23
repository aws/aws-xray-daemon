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

# Failures are recorded as "metric artifact region" lines, rather than
# associative arrays -- macOS still ships bash 3.2, which has no `declare -A`,
# and keeping this portable preserves local testability. Each region worker
# writes its OWN fail file ($WORK/fail.<region>) so parallel workers never
# append to a shared file concurrently; they are concatenated at the end.
# Counts and failing-region lists are then derived from the combined log.
MAX_PARALLEL="${MAX_PARALLEL:-8}"

record_dl_fail()  { echo "DownloadFailureFromS3 $1 $2" >> "$3"; }
record_sig_fail() { echo "SignatureVerificationFailureFromS3 $1 $2" >> "$3"; }

base_for() { echo "https://s3.$1.amazonaws.com/aws-xray-assets.$1/xray-daemon"; }

# curl -f exits non-zero on any HTTP error; this tests the real public customer
# download path (objects are public, no S3 read perms needed). Bounded timeouts
# keep a stalled S3 connection from holding a worker indefinitely -- with a
# 10-minute schedule that would let runs overlap without ever reporting a
# failure. --max-time is per attempt; --retry can add up to one more attempt.
download() { curl -fsSL --connect-timeout 10 --max-time 60 --retry 2 -o "$2" "$1"; }

# Availability-only check via HEAD -- confirms the object exists without pulling
# its bytes. Used for artifacts we don't verify locally (debs), which avoids
# downloading hundreds of MB of package data across all regions every run.
head_ok() { curl -fsSL --connect-timeout 10 --max-time 30 --retry 2 -o /dev/null -I "$1"; }

# Publish one metric point. In DRY_RUN mode, print instead of calling AWS so the
# script is runnable locally without credentials.
# Tracks whether any metric failed to publish. A failed publish must not leave
# the job green with missing data, but we also don't want to abort mid-run and
# skip the remaining metrics -- so failures are recorded and the script exits
# non-zero at the end (see the final PUBLISH_FAILED check).
PUBLISH_FAILED=0
emit() { # $1=metric-name  $2=value(count)  $3=artifact
  if [[ "$DRY_RUN" == "1" ]]; then
    echo "[dry-run] $1{artifact=$3} = $2"
    return 0
  fi
  if ! aws cloudwatch put-metric-data --metric-name "$1" --dimensions failure=rate,artifact="$3" --namespace MonitorDaemon --value "$2" --timestamp "$(date +%s)"; then
    echo "ERROR: failed to publish $1{artifact=$3}=$2" >&2
    PUBLISH_FAILED=1
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

echo "Checking ${#REGIONS[@]} region(s) (up to $MAX_PARALLEL in parallel); verification key import: $([[ $KEY_OK -eq 0 ]] && echo ok || echo FAILED)"

# Per-region worker: download every artifact + zip sigs, verify zip signatures
# with gpg (host), and record download/signature failures to this region's own
# fail file. RPM signature verification is NOT done here -- the downloaded rpms
# are left on disk and verified together in a single container afterward, so we
# spin up one Amazon Linux container total rather than one per region.
check_region() { # $1=region
  local region="$1"
  local BASE rdir ff name
  BASE="$(base_for "$region")"
  rdir="$WORK/$region"; mkdir -p "$rdir"
  ff="$WORK/fail.$region"; : > "$ff"

  download "$BASE/$KEY" "$rdir/$KEY" || record_dl_fail "$KEY" "$region" "$ff"

  for name in "${ZIP_ARTIFACTS[@]}"; do
    if download "$BASE/$name" "$rdir/$name" && download "$BASE/$name.sig" "$rdir/$name.sig"; then
      if [[ "$KEY_OK" -ne 0 ]] || ! gpg --verify "$rdir/$name.sig" "$rdir/$name" >/dev/null 2>&1; then
        record_sig_fail "$name" "$region" "$ff"
      fi
    else
      record_dl_fail "$name" "$region" "$ff"
      record_sig_fail "$name" "$region" "$ff"   # can't verify what didn't download
    fi
  done

  # rpms: download only here; a download failure also fails the signature (can't
  # verify what isn't present). Present rpms are verified in the container pass.
  for name in "${RPM_ARTIFACTS[@]}"; do
    download "$BASE/$name" "$rdir/$name" || { record_dl_fail "$name" "$region" "$ff"; record_sig_fail "$name" "$region" "$ff"; }
  done

  # debs: download availability only.
  # TODO: DEB signature verification uses the _gpgorigin member written by
  # Tool/src/packaging/sign-packages.sh. Verifying it robustly needs the exact
  # member concatenation order and is fragile enough to risk false alarms, so it
  # is deferred to its own change.
  for name in "${DEB_ARTIFACTS[@]}"; do
    head_ok "$BASE/$name" || record_dl_fail "$name" "$region" "$ff"
  done
}

# Fan out region workers, capped at MAX_PARALLEL concurrent jobs.
for region in "${REGIONS[@]}"; do
  check_region "$region" &
  while [[ "$(jobs -r | wc -l)" -ge "$MAX_PARALLEL" ]]; do wait -n 2>/dev/null || wait; done
done
wait

# ---- RPM signature verification: one Amazon Linux container for all regions ---
# rpm verification runs inside amazonlinux:2023, not on the ubuntu-latest host:
# modern Ubuntu's rpm uses the Sequoia backend, which rejects the daemon's
# (older RSA) signing key and would falsely report SIGNATURES NOT OK. Each
# region's rpms live in $WORK/<region>/; mount $WORK and verify them all at once.
# Mark every downloaded rpm as a signature failure. Used whenever the verifier
# cannot produce a trustworthy per-rpm verdict, so the check fails closed.
fail_all_rpm_sigs() {
  local region name
  for region in "${REGIONS[@]}"; do
    for name in "${RPM_ARTIFACTS[@]}"; do
      [[ -f "$WORK/$region/$name" ]] && record_sig_fail "$name" "$region" "$WORK/fail.$region"
    done
  done
}

if [[ "$KEY_OK" -ne 0 ]]; then
  # No usable key: every downloaded rpm's signature is a failure.
  fail_all_rpm_sigs
else
  cp verify-key.gpg "$WORK/verify-key.gpg"
  # The container prints "BAD <region>/<name>" for each rpm that fails
  # verification and "VERIFY_DONE" once it has checked every rpm. The sentinel
  # plus the docker exit code let us tell "verified, all good" apart from
  # "verifier never ran" (Docker daemon down, image-pull failure, etc.) -- the
  # latter must fail closed rather than emit healthy metrics.
  bad_rpms="$(docker run --rm -v "$WORK":/w -w /w amazonlinux:2023 bash -c '
    rpm --import verify-key.gpg >/dev/null 2>&1 || { echo IMPORT_FAILED; exit 0; }
    for f in */*.rpm; do
      [ -e "$f" ] || continue
      rpm -K "$f" | grep -q "signatures OK" || echo "BAD $f"
    done
    echo VERIFY_DONE' 2>/dev/null)"
  docker_rc=$?
  if [[ "$docker_rc" -ne 0 ]] || ! grep -q VERIFY_DONE <<< "$bad_rpms" || grep -q IMPORT_FAILED <<< "$bad_rpms"; then
    # Verifier could not run or could not import the key: fail closed.
    echo "WARN: rpm verifier did not complete (docker rc=$docker_rc); marking all rpm signatures failed"
    fail_all_rpm_sigs
  else
    # Each BAD line is "BAD <region>/<name>"; split back into region + artifact.
    while read -r _ path; do
      [[ -z "${path:-}" ]] && continue
      region="${path%%/*}"; name="${path##*/}"
      record_sig_fail "$name" "$region" "$WORK/fail.$region"
    done <<< "$(grep '^BAD ' <<< "$bad_rpms")"
  fi
fi

# Combine all per-region fail files into one log for aggregation.
FAILLOG="$WORK/failures.log"
cat "$WORK"/fail.* > "$FAILLOG" 2>/dev/null || : > "$FAILLOG"

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

# Fail the run if any metric could not be published -- otherwise a CloudWatch
# outage or credential problem would leave the job green with missing data.
if [[ "$PUBLISH_FAILED" -ne 0 ]]; then
  echo "ERROR: one or more metrics failed to publish" >&2
  exit 1
fi
