#!/usr/bin/env bash
echo "****************************************"
echo "Creating zip file for Windows amd64"
echo "****************************************"

BUILD_FOLDER=${BGO_SPACE}/build/xray

echo "Constructing the zip package"

if [ -f ${BUILD_FOLDER}/aws-xray-daemon-windows-service-`cat ${BGO_SPACE}/VERSION`.zip ]
then
    rm ${BUILD_FOLDER}/aws-xray-daemon-windows-service-`cat ${BGO_SPACE}/VERSION`.zip
fi

if [ -f ${BUILD_FOLDER}/aws-xray-daemon-windows-process-`cat ${BGO_SPACE}/VERSION`.zip ]
then
    rm ${BUILD_FOLDER}/aws-xray-daemon-windows-process-`cat ${BGO_SPACE}/VERSION`.zip
fi

cd ${BUILD_FOLDER}
zip aws-xray-daemon-windows-service-`cat ${BGO_SPACE}/VERSION`.zip xray.exe cfg.yaml
zip aws-xray-daemon-windows-process-`cat ${BGO_SPACE}/VERSION`.zip xray_windows.exe cfg.yaml
