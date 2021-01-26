#!/usr/bin/env bash
echo "****************************************"
echo "Creating zip file for Linux amd64 and arm64"
echo "****************************************"

DIST_FOLDER=${BGO_SPACE}/build/dist/
cd $DIST_FOLDER

cp ../xray-linux-amd64/xray xray
zip aws-xray-daemon-linux-amd64-${VERSION}.zip xray cfg.yaml LICENSE THIRD-PARTY-LICENSES.txt
rm xray

cp ../xray-linux-arm64/xray xray
zip aws-xray-daemon-linux-arm64-${VERSION}.zip xray cfg.yaml LICENSE THIRD-PARTY-LICENSES.txt
rm xray