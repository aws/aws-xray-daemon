#!/usr/bin/env bash
echo "****************************************"
echo "Creating legacy artifacts for 3.x with older names"
echo "****************************************"

cd ${BGO_SPACE}/build/dist

echo "Building and packaging legacy artifacts for Linux"
cp ${BGO_SPACE}/build/dist/aws-xray-daemon-linux-amd64-${VERSION}.zip ${BGO_SPACE}/build/dist/aws-xray-daemon-linux-3.x.zip
cp ${BGO_SPACE}/build/dist/aws-xray-daemon-linux-amd64-${VERSION}.rpm ${BGO_SPACE}/build/dist/aws-xray-daemon-3.x.rpm
cp ${BGO_SPACE}/build/dist/aws-xray-daemon-linux-amd64-${VERSION}.deb ${BGO_SPACE}/build/dist/aws-xray-daemon-3.x.deb
cp ${BGO_SPACE}/build/dist/aws-xray-daemon-linux-arm64-${VERSION}.rpm ${BGO_SPACE}/build/dist/aws-xray-daemon-arm64-3.x.rpm
cp ${BGO_SPACE}/build/dist/aws-xray-daemon-linux-arm64-${VERSION}.deb ${BGO_SPACE}/build/dist/aws-xray-daemon-arm64-3.x.deb

echo "Building and packaging legacy artifacts for MacOS"
GOOS=darwin GOARCH=amd64 go build -ldflags "-s -w" -o ${BGO_SPACE}/build/xray-mac-legacy/xray_mac ${PREFIX}/cmd/tracing/daemon.go ${PREFIX}/cmd/tracing/tracing.go
cp ${BGO_SPACE}/build/xray-mac-legacy/xray_mac xray_mac
zip aws-xray-daemon-macos-3.x.zip xray_mac cfg.yaml LICENSE THIRD-PARTY-LICENSES.txt
rm xray_mac

echo "Building and packaging legacy artifacts for Windows"
GOOS=windows GOARCH=amd64 go build -ldflags "-s -w" -o ${BGO_SPACE}/build/xray-win-legacy/xray.exe ${PREFIX}/cmd/tracing/daemon.go ${PREFIX}/cmd/tracing/tracing_windows.go
GOOS=windows GOARCH=amd64 go build -ldflags "-s -w" -o ${BGO_SPACE}/build/xray-win-legacy/xray_windows.exe ${PREFIX}/cmd/tracing/daemon.go ${PREFIX}/cmd/tracing/tracing.go
cp ${BGO_SPACE}/build/xray-win-legacy/xray.exe xray.exe
zip aws-xray-daemon-windows-service-3.x.zip xray.exe cfg.yaml LICENSE THIRD-PARTY-LICENSES.txt
rm xray.exe
cp ${BGO_SPACE}/build/xray-win-legacy/xray_windows.exe xray_windows.exe
zip aws-xray-daemon-windows-process-3.x.zip xray_windows.exe cfg.yaml LICENSE THIRD-PARTY-LICENSES.txt
rm xray_windows.exe

rm cfg.yaml LICENSE THIRD-PARTY-LICENSES.txt