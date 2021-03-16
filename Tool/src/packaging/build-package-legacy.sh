#!/usr/bin/env bash

set -e

echo "****************************************"
echo "Creating legacy artifacts for 3.x with older names"
echo "****************************************"

cd ${BGO_SPACE}/build/dist

echo "Building and packaging legacy artifacts for Linux"
cp ${BGO_SPACE}/build/dist/aws-xray-daemon-linux-amd64-${VERSION}.zip ${BGO_SPACE}/build/dist/aws-xray-daemon-linux-3.x.zip
cp ${BGO_SPACE}/build/dist/aws-xray-daemon-linux-amd64-${VERSION}.rpm ${BGO_SPACE}/build/dist/aws-xray-daemon-3.x.rpm
cp ${BGO_SPACE}/build/dist/aws-xray-daemon-linux-amd64-${VERSION}.deb ${BGO_SPACE}/build/dist/aws-xray-daemon-3.x.deb
cp ${BGO_SPACE}/build/dist/aws-xray-daemon-linux-arm64-${VERSION}.zip ${BGO_SPACE}/build/dist/aws-xray-daemon-linux-arm64-3.x.zip
cp ${BGO_SPACE}/build/dist/aws-xray-daemon-linux-arm64-${VERSION}.rpm ${BGO_SPACE}/build/dist/aws-xray-daemon-arm64-3.x.rpm
cp ${BGO_SPACE}/build/dist/aws-xray-daemon-linux-arm64-${VERSION}.deb ${BGO_SPACE}/build/dist/aws-xray-daemon-arm64-3.x.deb

echo "Building and packaging legacy artifacts for MacOS"
unzip -q -o aws-xray-daemon-macos-amd64-${VERSION}.zip
mv xray xray_mac
zip aws-xray-daemon-macos-3.x.zip xray_mac cfg.yaml LICENSE THIRD-PARTY-LICENSES.txt
rm xray_mac

echo "Building and packaging legacy artifacts for Windows"
unzip -q -o aws-xray-daemon-windows-amd64-service-${VERSION}.zip
mv xray_service.exe xray.exe
zip aws-xray-daemon-windows-service-3.x.zip xray.exe cfg.yaml LICENSE THIRD-PARTY-LICENSES.txt
rm xray.exe

unzip -q -o aws-xray-daemon-windows-amd64-${VERSION}.zip
mv xray.exe xray_windows.exe
zip aws-xray-daemon-windows-process-3.x.zip xray_windows.exe cfg.yaml LICENSE THIRD-PARTY-LICENSES.txt
rm xray_windows.exe

rm cfg.yaml LICENSE THIRD-PARTY-LICENSES.txt
