#!/usr/bin/env bash
echo "****************************************"
echo "Creating zip file for Linux amd64"
echo "****************************************"

BUILD_FOLDER=${BGO_SPACE}/build/xray

if [ -f ${BUILD_FOLDER}/xray-linux.zip ]
then
    rm ${BUILD_FOLDER}/xray-linux.zip
fi
cd ${BUILD_FOLDER}
zip aws-xray-daemon-linux-`cat ${BGO_SPACE}/VERSION`.zip xray cfg.yaml LICENSE THIRD-PARTY-LICENSES.txt
