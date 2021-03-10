#!/bin/sh

BUILD_FOLDER=${TARGETPLATFORM/\//-}

echo "Building for platform ${TARGETPLATFORM}"

if [[ -d "/workspace/build/xray-${BUILD_FOLDER}" ]]; then
  echo "Copying prebuilt binary"
  cp /workspace/build/xray-${BUILD_FOLDER}/xray /workspace/
else
  echo "Building from source"
  CGO_ENABLED=0 GOOS=linux GOARCH=$(echo $TARGETPLATFORM | cut -d'/' -f2) go build -a -ldflags '-extldflags "-static"' -o /workspace/xray /workspace/cmd/tracing
fi
