#!/usr/bin/env bash

set -e

ARCH=$1
if [[ $ARCH == "amd64" ]]
then
    BUILD_ARCH=x86_64
elif [[ $ARCH == "arm64" ]]
then
    BUILD_ARCH=aarch64
else
    echo "Invalid architecture input. Exiting"
    exit 1
fi

echo "*************************************************"
echo "Creating rpm file for Amazon Linux and RHEL ${ARCH}"
echo "*************************************************"

rm -rf ${BGO_SPACE}/bin/linux_${ARCH}/linux

echo "Creating rpmbuild workspace"
mkdir -p ${BGO_SPACE}/bin/linux_${ARCH}/linux/rpmbuild/{RPMS,SRPMS,BUILD,COORD_SOURCES,SPECS,DATA_SOURCES}
mkdir -p ${BGO_SPACE}/bin/linux_${ARCH}/linux/usr/bin/
mkdir -p ${BGO_SPACE}/bin/linux_${ARCH}/linux/etc/amazon/xray/
mkdir -p ${BGO_SPACE}/bin/linux_${ARCH}/linux/etc/init/
mkdir -p ${BGO_SPACE}/bin/linux_${ARCH}/linux/etc/systemd/system/

echo "Copying application files"
cp ${BGO_SPACE}/build/xray-linux-${ARCH}/xray ${BGO_SPACE}/bin/linux_${ARCH}/linux/usr/bin/
cp ${BGO_SPACE}/pkg/cfg.yaml ${BGO_SPACE}/bin/linux_${ARCH}/linux/etc/amazon/xray/cfg.yaml
cp ${BGO_SPACE}/Tool/src/packaging/linux/xray.conf ${BGO_SPACE}/bin/linux_${ARCH}/linux/etc/init/
cp ${BGO_SPACE}/Tool/src/packaging/linux/xray.service ${BGO_SPACE}/bin/linux_${ARCH}/linux/etc/systemd/system/
cp ${BGO_SPACE}/LICENSE ${BGO_SPACE}/bin/linux_${ARCH}/linux/etc/amazon/xray/
cp ${BGO_SPACE}/THIRD-PARTY-LICENSES.txt ${BGO_SPACE}/bin/linux_${ARCH}/linux/etc/amazon/xray/

echo "Creating the rpm package"
SPEC_FILE="${BGO_SPACE}/Tool/src/packaging/linux/xray.spec"
BUILD_ROOT="${BGO_SPACE}/bin/linux_${ARCH}/linux"
rpmbuild --target ${BUILD_ARCH}-linux --define "rpmversion ${VERSION}" --define "_topdir bin/linux_${ARCH}/linux/rpmbuild" -bb --buildroot ${BUILD_ROOT} ${SPEC_FILE}

echo "Copying rpm files to bin"
cp ${BGO_SPACE}/bin/linux_${ARCH}/linux/rpmbuild/RPMS/${BUILD_ARCH}/*.rpm ${BGO_SPACE}/build/dist/aws-xray-daemon-linux-${ARCH}-${VERSION}.rpm
