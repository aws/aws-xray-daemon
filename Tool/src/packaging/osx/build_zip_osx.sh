#!/usr/bin/env bash
echo "****************************************"
echo "Creating zip file for OS-X amd64"
echo "****************************************"

BUILD_FOLDER=${BGO_SPACE}/build/xray

if [ -f ${BUILD_FOLDER}/xray-osx.zip ]
then
    rm ${BUILD_FOLDER}/xray-osx.zip
fi
cd ${BUILD_FOLDER}
zip aws-xray-daemon-macos-`cat ${BGO_SPACE}/VERSION`.zip xray_mac cfg.yaml
