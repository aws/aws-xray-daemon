#!/usr/bin/env bash
echo "****************************************"
echo "Creating zip file for Linux amd64 and arm64"
echo "****************************************"

DIST_FOLDER=${BGO_SPACE}/build/dist/
cd $DIST_FOLDER

zip aws-xray-daemon-linux-amd64-${VERSION}.zip ../xray-linux-amd64/xray ../xray/cfg.yaml ../xray/LICENSE ../xray/THIRD-PARTY-LICENSES.txt
zip aws-xray-daemon-linux-arm64-${VERSION}.zip ../xray-linux-arm64/xray ../xray/cfg.yaml ../xray/LICENSE ../xray/THIRD-PARTY-LICENSES.txt
