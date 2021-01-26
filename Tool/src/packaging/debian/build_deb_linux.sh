#!/usr/bin/env bash

ARCH=$1

echo "****************************************"
echo "Creating deb file for Ubuntu Linux ${ARCH}"
echo "****************************************"

echo "Creating debian folders"

mkdir -p ${BGO_SPACE}/bin/debian_${ARCH}/debian/usr/bin/
mkdir -p ${BGO_SPACE}/bin/debian_${ARCH}/debian/etc/init/
mkdir -p ${BGO_SPACE}/bin/debian_${ARCH}/debian/etc/amazon/xray/
mkdir -p ${BGO_SPACE}/bin/debian_${ARCH}/debian/var/lib/amazon/xray/
mkdir -p ${BGO_SPACE}/bin/debian_${ARCH}/debian/lib/systemd/system/
mkdir -p ${BGO_SPACE}/bin/debian_${ARCH}/debian/usr/share/lintian/overrides/
mkdir -p ${BGO_SPACE}/bin/debian_${ARCH}/debian/usr/share/doc/xray/

echo "Copying application files"

cp ${BGO_SPACE}/build/xray-linux-${ARCH}/xray ${BGO_SPACE}/bin/debian_${ARCH}/debian/usr/bin/
cp ${BGO_SPACE}/build/dist/cfg.yaml ${BGO_SPACE}/bin/debian_${ARCH}/debian/etc/amazon/xray/cfg.yaml
cp ${BGO_SPACE}/Tool/src/packaging/debian/xray.conf ${BGO_SPACE}/bin/debian_${ARCH}/debian/etc/init/xray.conf
cp ${BGO_SPACE}/Tool/src/packaging/debian/xray.service ${BGO_SPACE}/bin/debian_${ARCH}/debian/lib/systemd/system/xray.service

echo "Copying debian package config files"

cp ${BGO_SPACE}/Tool/src/packaging/debian/conffiles ${BGO_SPACE}/bin/debian_${ARCH}/debian/
cp ${BGO_SPACE}/Tool/src/packaging/debian/preinst ${BGO_SPACE}/bin/debian_${ARCH}/debian/
cp ${BGO_SPACE}/Tool/src/packaging/debian/postinst ${BGO_SPACE}/bin/debian_${ARCH}/debian/
cp ${BGO_SPACE}/Tool/src/packaging/debian/prerm ${BGO_SPACE}/bin/debian_${ARCH}/debian/
cp ${BGO_SPACE}/Tool/src/packaging/debian/lintian-overrides ${BGO_SPACE}/bin/debian_${ARCH}/debian/usr/share/lintian/overrides/xray

echo "Constructing the control file"

echo "Package: xray" > ${BGO_SPACE}/bin/debian_${ARCH}/debian/control
echo "Architecture: ${ARCH}" >> ${BGO_SPACE}/bin/debian_${ARCH}/debian/control
echo -n "Version: " >> ${BGO_SPACE}/bin/debian_${ARCH}/debian/control
echo $VERSION >> ${BGO_SPACE}/bin/debian_${ARCH}/debian/control
echo "-1" >> ${BGO_SPACE}/bin/debian_${ARCH}/debian/control
cat ${BGO_SPACE}/Tool/src/packaging/debian/control >> ${BGO_SPACE}/bin/debian_${ARCH}/debian/control

echo "Constructing the copyright file"
cat ${BGO_SPACE}/LICENSE >> ${BGO_SPACE}/bin/debian_${ARCH}/debian/copyright
echo "\n ======================== \n" >> ${BGO_SPACE}/bin/debian_${ARCH}/debian/copyright
cat ${BGO_SPACE}/THIRD-PARTY-LICENSES.txt >> ${BGO_SPACE}/bin/debian_${ARCH}/debian/copyright

echo "Constructing the changelog file"

echo -n "xray (" > ${BGO_SPACE}/bin/debian_${ARCH}/debian/usr/share/doc/xray/changelog
echo $VERSION >> ${BGO_SPACE}/bin/debian_${ARCH}/debian/usr/share/doc/xray/changelog
echo "-1) precise-proposed; urgency=low" >> ${BGO_SPACE}/bin/debian_${ARCH}/debian/usr/share/doc/xray/changelog
cat ${BGO_SPACE}/Tool/src/packaging/debian/changelog >> ${BGO_SPACE}/bin/debian_${ARCH}/debian/usr/share/doc/xray/changelog

cp ${BGO_SPACE}/Tool/src/packaging/debian/changelog.Debian ${BGO_SPACE}/bin/debian_${ARCH}/debian/usr/share/doc/xray/
cp ${BGO_SPACE}/Tool/src/packaging/debian/debian-binary ${BGO_SPACE}/bin/debian_${ARCH}/debian/

echo "Setting permissioning as required by debian"

cd ${BGO_SPACE}/bin/debian_${ARCH}/; find ./debian -type d | xargs chmod 755; cd ~-

echo "Compressing changelog"

cd ${BGO_SPACE}/bin/debian_${ARCH}/debian/usr/share/doc/xray/; export GZIP=-9; tar cvzf changelog.gz changelog --owner=0 --group=0 ; cd ~-
cd ${BGO_SPACE}/bin/debian_${ARCH}/debian/usr/share/doc/xray/; export GZIP=-9; tar cvzf changelog.Debian.gz changelog.Debian --owner=0 --group=0; cd ~-

rm ${BGO_SPACE}/bin/debian_${ARCH}/debian/usr/share/doc/xray/changelog
rm ${BGO_SPACE}/bin/debian_${ARCH}/debian/usr/share/doc/xray/changelog.Debian

echo "Creating tar"
# the below permission is required by debian
cd ${BGO_SPACE}/bin/debian_${ARCH}/debian/; tar czf data.tar.gz usr etc lib --owner=0 --group=0 ; cd ~-
cd ${BGO_SPACE}/bin/debian_${ARCH}/debian/; tar czf control.tar.gz control conffiles preinst postinst prerm copyright --owner=0 --group=0 ; cd ~-

echo "Constructing the deb package"
ar r ${BGO_SPACE}/bin/xray.deb ${BGO_SPACE}/bin/debian_${ARCH}/debian/debian-binary
ar r ${BGO_SPACE}/bin/xray.deb ${BGO_SPACE}/bin/debian_${ARCH}/debian/control.tar.gz
ar r ${BGO_SPACE}/bin/xray.deb ${BGO_SPACE}/bin/debian_${ARCH}/debian/data.tar.gz
cp ${BGO_SPACE}/bin/xray.deb ${BGO_SPACE}/build/dist/aws-xray-daemon-linux-${ARCH}-${VERSION}.deb
