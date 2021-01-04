#!/usr/bin/env bash
echo "*************************************************"
echo "Creating rpm file for Amazon Linux and RHEL amd64"
echo "*************************************************"

rm -rf ${BGO_SPACE}/bin/linux_amd64/linux

echo "Creating rpmbuild workspace"
mkdir -p ${BGO_SPACE}/bin/linux_amd64/linux/rpmbuild/{RPMS,SRPMS,BUILD,COORD_SOURCES,SPECS,DATA_SOURCES}
mkdir -p ${BGO_SPACE}/bin/linux_amd64/linux/usr/bin/
mkdir -p ${BGO_SPACE}/bin/linux_amd64/linux/etc/amazon/xray/
mkdir -p ${BGO_SPACE}/bin/linux_amd64/linux/etc/init/
mkdir -p ${BGO_SPACE}/bin/linux_amd64/linux/etc/systemd/system/

echo "Copying application files"
cp ${BGO_SPACE}/build/xray-linux-amd64/xray ${BGO_SPACE}/bin/linux_amd64/linux/usr/bin/
cp ${BGO_SPACE}/pkg/cfg.yaml ${BGO_SPACE}/bin/linux_amd64/linux/etc/amazon/xray/cfg.yaml
cp ${BGO_SPACE}/Tool/src/packaging/linux/xray.conf ${BGO_SPACE}/bin/linux_amd64/linux/etc/init/
cp ${BGO_SPACE}/Tool/src/packaging/linux/xray.service ${BGO_SPACE}/bin/linux_amd64/linux/etc/systemd/system/
cp ${BGO_SPACE}/LICENSE ${BGO_SPACE}/bin/linux_amd64/linux/etc/amazon/xray/
cp ${BGO_SPACE}/THIRD-PARTY-LICENSES.txt ${BGO_SPACE}/bin/linux_amd64/linux/etc/amazon/xray/

echo "Creating the rpm package"
SPEC_FILE="${BGO_SPACE}/Tool/src/packaging/linux/xray.spec"
BUILD_ROOT="${BGO_SPACE}/bin/linux_amd64/linux"
setarch x86_64 rpmbuild --define "rpmversion ${VERSION}" --define "_topdir bin/linux_amd64/linux/rpmbuild" -bb --buildroot ${BUILD_ROOT} ${SPEC_FILE}

echo "Copying rpm files to bin"
cp ${BGO_SPACE}/bin/linux_amd64/linux/rpmbuild/RPMS/x86_64/*.rpm ${BGO_SPACE}/build/dist/aws-xray-daemon-linux-amd64-${VERSION}.rpm
