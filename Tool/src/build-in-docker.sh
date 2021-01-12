#!/bin/sh

TARGETPLATFORM=${TARGETPLATFORM:-linux/amd64}
BUILD_FOLDER=${TARGETPLATFORM/\//-}

if [[ -d "/workspace/build/xray-${BUILD_FOLDER}" ]]; then
  echo "Copying prebuilt binary"
  cp /workspace/build/xray-${BUILD_FOLDER}/xray /workspace
else
  echo "Building from source"
  CGO_ENABLED=0 GOOS=linux GOARCH=$(echo $TARGETPLATFORM | cut -d'/' -f2) go build -a -ldflags '-extldflags "-static"' -o /workspace/xray /workspace/cmd/tracing
fi
