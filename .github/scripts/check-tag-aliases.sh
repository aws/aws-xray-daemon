#!/usr/bin/env bash
#
# Tag/alias freshness check across the daemon's distribution surfaces.
# Invoked by the tag-alias-check job in continuous-monitoring.yml.
#
# The docs tell customers to use the moving aliases (`latest`, `3.x`). If an
# alias goes stale -- still pointing at an older release after a new one ships
# -- customers silently get the wrong version. This verifies each alias
# resolves to the SAME content as the explicit current-version tag:
#   - ECR / Docker Hub: image manifest digest of latest and 3.x == digest of <version>
#   - S3:               ETag of the 3.x zip == ETag of the amd64-<version> zip
#
# The expected version is taken from the latest GitHub release, so the check
# tracks releases automatically with no hardcoded version.
#
# Publishes to the MonitorDaemon namespace (failure=rate dimension, 0 ok / 1
# stale-or-error), with a channel dimension:
#   - TagAliasMismatchFromECR
#   - TagAliasMismatchFromDockerhub
#   - TagAliasMismatchFromS3
#
# Environment:
#   VERSION   override the expected version (e.g. 3.6.6). Default: derived from
#             the latest GitHub release tag.
#   DRY_RUN   if 1, print metrics instead of calling CloudWatch (no creds).
set -uo pipefail

DRY_RUN="${DRY_RUN:-0}"

emit() { # $1=metric-name  $2=value(0|1)  $3=channel
  if [[ "$DRY_RUN" == "1" ]]; then
    echo "[dry-run] $1{channel=$3} = $2"
  else
    aws cloudwatch put-metric-data --metric-name "$1" --dimensions failure=rate,channel="$3" --namespace MonitorDaemon --value "$2" --timestamp "$(date +%s)"
  fi
}

# ---- Determine the expected version -----------------------------------------
VERSION="${VERSION:-}"
if [[ -z "$VERSION" ]]; then
  # Latest release tag looks like v3.6.6; strip the leading v.
  VERSION="$(curl -fsSL "https://api.github.com/repos/aws/aws-xray-daemon/releases/latest" \
    | grep -m1 '"tag_name"' | sed -E 's/.*"v?([0-9]+\.[0-9]+\.[0-9]+)".*/\1/')"
fi
if [[ ! "$VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "Could not determine expected version (got: '${VERSION:-}')" >&2
  # Can't check anything meaningfully; report all three channels as failed.
  emit TagAliasMismatchFromECR 1 "3.x-and-latest"
  emit TagAliasMismatchFromDockerhub 1 "3.x-and-latest"
  emit TagAliasMismatchFromS3 1 "3.x"
  exit 0
fi
echo "Expected version: $VERSION"

# ---- Registry manifest digest -----------------------------------------------
# Returns the docker-content-digest header for a tag, or empty on error.
registry_digest() { # $1=registry-host  $2=repo-path  $3=tag  $4=bearer-token(optional)
  local host="$1" repo="$2" tag="$3" token="${4:-}"
  local auth=()
  [[ -n "$token" ]] && auth=(-H "Authorization: Bearer $token")
  curl -fsSL -I "${auth[@]}" \
    -H "Accept: application/vnd.docker.distribution.manifest.v2+json" \
    -H "Accept: application/vnd.oci.image.index.v1+json" \
    -H "Accept: application/vnd.docker.distribution.manifest.list.v2+json" \
    "https://$host/v2/$repo/manifests/$tag" 2>/dev/null \
    | grep -i "docker-content-digest" | tr -d '\r' | awk '{print $2}'
}

# Compares the digests of the given aliases against the version tag; emits the
# metric 1 if any alias is missing or mismatched, else 0.
check_registry() { # $1=metric  $2=channel  $3=host  $4=repo  $5=token  ...aliases
  local metric="$1" channel="$2" host="$3" repo="$4" token="$5"; shift 5
  local want alias d bad=0
  want="$(registry_digest "$host" "$repo" "$VERSION" "$token")"
  if [[ -z "$want" ]]; then
    echo "  [$channel] could not read digest for version $VERSION"; emit "$metric" 1 "$channel"; return
  fi
  for alias in "$@"; do
    d="$(registry_digest "$host" "$repo" "$alias" "$token")"
    if [[ "$d" != "$want" ]]; then
      echo "  [$channel] MISMATCH: $alias=$d != $VERSION=$want"; bad=1
    else
      echo "  [$channel] ok: $alias == $VERSION"
    fi
  done
  emit "$metric" "$bad" "$channel"
}

# ECR public (anonymous bearer token).
ECR_TOKEN="$(curl -fsSL "https://public.ecr.aws/token/" | sed -E 's/.*"token":"([^"]+)".*/\1/')"
check_registry TagAliasMismatchFromECR ecr public.ecr.aws xray/aws-xray-daemon "$ECR_TOKEN" latest 3.x

# Docker Hub (anonymous pull token, scoped to the repo).
DH_TOKEN="$(curl -fsSL "https://auth.docker.io/token?service=registry.docker.io&scope=repository:amazon/aws-xray-daemon:pull" | sed -E 's/.*"token":"([^"]+)".*/\1/')"
check_registry TagAliasMismatchFromDockerhub dockerhub registry-1.docker.io amazon/aws-xray-daemon "$DH_TOKEN" latest 3.x

# ---- S3: ETag of the 3.x zip vs the amd64-<version> zip ----------------------
# Post-3.3.0 the versioned object name embeds the arch, e.g.
# aws-xray-daemon-linux-amd64-3.6.6.zip, while the moving alias omits it for
# amd64: aws-xray-daemon-linux-3.x.zip. A matching ETag proves the alias points
# at the current release.
s3_etag() { curl -fsSL -I "$1" 2>/dev/null | grep -i "^etag:" | tr -d '\r' | awk '{print $2}'; }

S3BASE="https://s3.us-east-2.amazonaws.com/aws-xray-assets.us-east-2/xray-daemon"
alias_etag="$(s3_etag "$S3BASE/aws-xray-daemon-linux-3.x.zip")"
ver_etag="$(s3_etag "$S3BASE/aws-xray-daemon-linux-amd64-$VERSION.zip")"
if [[ -z "$alias_etag" || -z "$ver_etag" ]]; then
  echo "  [s3] could not read one or both ETags (alias=$alias_etag ver=$ver_etag)"
  emit TagAliasMismatchFromS3 1 "3.x"
elif [[ "$alias_etag" == "$ver_etag" ]]; then
  echo "  [s3] ok: 3.x == $VERSION ($alias_etag)"
  emit TagAliasMismatchFromS3 0 "3.x"
else
  echo "  [s3] MISMATCH: 3.x=$alias_etag != $VERSION=$ver_etag"
  emit TagAliasMismatchFromS3 1 "3.x"
fi
