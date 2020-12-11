#!/usr/bin/env bash
echo "****************************************"
echo "Creating zip file for Windows amd64"
echo "****************************************"

DIST_FOLDER=${BGO_SPACE}/build/dist/
cd $DIST_FOLDER

zip aws-xray-daemon-windows-amd64-service-`cat ${BGO_SPACE}/VERSION`.zip ../xray-windows-amd64/xray_service.exe ../xray/cfg.yaml ../xray/LICENSE ../xray/THIRD-PARTY-LICENSES.txt
zip aws-xray-daemon-windows-amd64-`cat ${BGO_SPACE}/VERSION`.zip ../xray-windows-amd64/xray.exe ../xray/cfg.yaml ../xray/LICENSE ../xray/THIRD-PARTY-LICENSES.txt
