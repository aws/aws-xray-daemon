#!/usr/bin/env bash
echo "****************************************"
echo "Creating deb file for Ubuntu Linux amd64"
echo "****************************************"

echo "Creating debian folders"

mkdir -p ${BGO_SPACE}/bin/debian_amd64/debian/usr/bin/
mkdir -p ${BGO_SPACE}/bin/debian_amd64/debian/etc/init/
mkdir -p ${BGO_SPACE}/bin/debian_amd64/debian/etc/amazon/xray/
mkdir -p ${BGO_SPACE}/bin/debian_amd64/debian/var/lib/amazon/xray/
mkdir -p ${BGO_SPACE}/bin/debian_amd64/debian/lib/systemd/system/
mkdir -p ${BGO_SPACE}/bin/debian_amd64/debian/usr/share/lintian/overrides/
mkdir -p ${BGO_SPACE}/bin/debian_amd64/debian/usr/share/doc/xray/

echo "Copying application files"

cp ${BGO_SPACE}/build/xray-linux-amd64/xray ${BGO_SPACE}/bin/debian_amd64/debian/usr/bin/
cp ${BGO_SPACE}/build/xray/cfg.yaml ${BGO_SPACE}/bin/debian_amd64/debian/etc/amazon/xray/cfg.yaml
cp ${BGO_SPACE}/Tool/src/packaging/debian/xray.conf ${BGO_SPACE}/bin/debian_amd64/debian/etc/init/xray.conf
cp ${BGO_SPACE}/Tool/src/packaging/debian/xray.service ${BGO_SPACE}/bin/debian_amd64/debian/lib/systemd/system/xray.service

echo "Copying debian package config files"

cp ${BGO_SPACE}/Tool/src/packaging/debian/conffiles ${BGO_SPACE}/bin/debian_amd64/debian/
cp ${BGO_SPACE}/Tool/src/packaging/debian/preinst ${BGO_SPACE}/bin/debian_amd64/debian/
cp ${BGO_SPACE}/Tool/src/packaging/debian/postinst ${BGO_SPACE}/bin/debian_amd64/debian/
cp ${BGO_SPACE}/Tool/src/packaging/debian/prerm ${BGO_SPACE}/bin/debian_amd64/debian/
cp ${BGO_SPACE}/Tool/src/packaging/debian/lintian-overrides ${BGO_SPACE}/bin/debian_amd64/debian/usr/share/lintian/overrides/xray

echo "Constructing the control file"

echo 'Package: xray' > ${BGO_SPACE}/bin/debian_amd64/debian/control
echo 'Architecture: amd64' >> ${BGO_SPACE}/bin/debian_amd64/debian/control
echo -n 'Version: ' >> ${BGO_SPACE}/bin/debian_amd64/debian/control
cat ${BGO_SPACE}/VERSION | tr -d "\n" >> ${BGO_SPACE}/bin/debian_amd64/debian/control
echo '-1' >> ${BGO_SPACE}/bin/debian_amd64/debian/control
cat ${BGO_SPACE}/Tool/src/packaging/debian/control >> ${BGO_SPACE}/bin/debian_amd64/debian/control

echo "Constructing the copyright file"
cat ${BGO_SPACE}/LICENSE >> ${BGO_SPACE}/bin/debian_amd64/debian/copyright
echo '\n ======================== \n' >> ${BGO_SPACE}/bin/debian_amd64/debian/copyright
cat ${BGO_SPACE}/THIRD-PARTY-LICENSES.txt >> ${BGO_SPACE}/bin/debian_amd64/debian/copyright

echo "Constructing the changelog file"

echo -n 'xray (' > ${BGO_SPACE}/bin/debian_amd64/debian/usr/share/doc/xray/changelog
cat VERSION | tr -d "\n"  >> ${BGO_SPACE}/bin/debian_amd64/debian/usr/share/doc/xray/changelog
echo '-1) precise-proposed; urgency=low' >> ${BGO_SPACE}/bin/debian_amd64/debian/usr/share/doc/xray/changelog
cat ${BGO_SPACE}/Tool/src/packaging/debian/changelog >> ${BGO_SPACE}/bin/debian_amd64/debian/usr/share/doc/xray/changelog

cp ${BGO_SPACE}/Tool/src/packaging/debian/changelog.Debian ${BGO_SPACE}/bin/debian_amd64/debian/usr/share/doc/xray/
cp ${BGO_SPACE}/Tool/src/packaging/debian/debian-binary ${BGO_SPACE}/bin/debian_amd64/debian/

echo "Setting permissioning as required by debian"

cd ${BGO_SPACE}/bin/debian_amd64/; find ./debian -type d | xargs chmod 755; cd ~-

echo "Compressing changelog"

cd ${BGO_SPACE}/bin/debian_amd64/debian/usr/share/doc/xray/; export GZIP=-9; tar cvzf changelog.gz changelog --owner=0 --group=0 ; cd ~-
cd ${BGO_SPACE}/bin/debian_amd64/debian/usr/share/doc/xray/; export GZIP=-9; tar cvzf changelog.Debian.gz changelog.Debian --owner=0 --group=0; cd ~-

rm ${BGO_SPACE}/bin/debian_amd64/debian/usr/share/doc/xray/changelog
rm ${BGO_SPACE}/bin/debian_amd64/debian/usr/share/doc/xray/changelog.Debian

echo "Creating tar"
# the below permission is required by debian
cd ${BGO_SPACE}/bin/debian_amd64/debian/; tar czf data.tar.gz usr etc lib --owner=0 --group=0 ; cd ~-
cd ${BGO_SPACE}/bin/debian_amd64/debian/; tar czf control.tar.gz control conffiles preinst postinst prerm copyright --owner=0 --group=0 ; cd ~-

echo "Constructing the deb package"
ar r ${BGO_SPACE}/bin/xray.deb ${BGO_SPACE}/bin/debian_amd64/debian/debian-binary
ar r ${BGO_SPACE}/bin/xray.deb ${BGO_SPACE}/bin/debian_amd64/debian/control.tar.gz
ar r ${BGO_SPACE}/bin/xray.deb ${BGO_SPACE}/bin/debian_amd64/debian/data.tar.gz
cp ${BGO_SPACE}/bin/xray.deb ${BGO_SPACE}/build/dist/aws-xray-daemon-linux-amd64-`cat ${BGO_SPACE}/VERSION`.deb
