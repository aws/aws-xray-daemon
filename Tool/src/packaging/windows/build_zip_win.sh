#!/usr/bin/env bash
echo "****************************************"
echo "Creating zip file for Windows amd64"
echo "****************************************"

DIST_FOLDER=${BGO_SPACE}/build/dist/
cd $DIST_FOLDER

cp ../xray-windows-amd64/xray_service.exe xray_service.exe
zip aws-xray-daemon-windows-amd64-service-${VERSION}.zip xray_service.exe cfg.yaml LICENSE THIRD-PARTY-LICENSES.txt
rm xray_service.exe

cp ../xray-windows-amd64/xray.exe xray.exe
zip aws-xray-daemon-windows-amd64-${VERSION}.zip xray.exe cfg.yaml LICENSE THIRD-PARTY-LICENSES.txt
rm xray.exe
