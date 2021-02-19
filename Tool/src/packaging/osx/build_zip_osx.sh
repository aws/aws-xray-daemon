#!/usr/bin/env bash
echo "****************************************"
echo "Creating zip file for OS-X amd64"
echo "****************************************"

DIST_FOLDER=${BGO_SPACE}/build/dist/
cd $DIST_FOLDER

cp ../xray-mac-amd64/xray xray
zip aws-xray-daemon-macos-amd64-${VERSION}.zip xray cfg.yaml LICENSE THIRD-PARTY-LICENSES.txt
rm xray

cp ../xray-mac-arm64/xray xray
zip aws-xray-daemon-macos-arm64-${VERSION}.zip xray cfg.yaml LICENSE THIRD-PARTY-LICENSES.txt
rm xray
