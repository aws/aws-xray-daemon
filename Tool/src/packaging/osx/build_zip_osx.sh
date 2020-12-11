#!/usr/bin/env bash
echo "****************************************"
echo "Creating zip file for OS-X amd64"
echo "****************************************"

DIST_FOLDER=${BGO_SPACE}/build/dist/
cd $DIST_FOLDER

zip aws-xray-daemon-macos-amd64-`cat ${BGO_SPACE}/VERSION`.zip ../xray-mac-amd64/xray cfg.yaml ../xray/LICENSE ../xray/THIRD-PARTY-LICENSES.txt
