#!/usr/bin/env bash

echo "****************************************"
echo "GPG signing for daemon assets"
echo "****************************************"

echo "Starting gpg signing for zip files"
for filename in build/dist/*; do
  ext="${filename##*.}"
  if [ $ext == "zip" ]; then
    gpg --output $filename.sig --detach-sig $filename
  fi
done
echo "Finished gpg signing for zip files"


echo "Starting GPG signing for rpm files"
rpm -qa gpg-pubkey

echo "Create rpmmacros file"
rm ~/.rpmmacros
echo -e "%_signature gpg\n%_gpg_path ~/.gnupg\n%_gpg_name AWS X-Ray\n%_gpgbin /usr/bin/gpg" >> ~/.rpmmacros
cat ~/.rpmmacros

for filename in build/dist/*; do
  ext="${filename##*.}"
  if [ $ext == "rpm" ]; then
    rpmsign --addsign $filename
  fi
done
echo "Finished GPG signing for rpm files"

echo "Starting GPG signing for deb files"
for filename in build/dist/*; do
  ext="${filename##*.}"
  if [ $ext == "deb" ]; then
    ar x $filename
    cat debian-binary control.tar.gz data.tar.gz > /tmp/combined-contents
    gpg -abs -o _gpgorigin /tmp/combined-contents
    ar rc $filename _gpgorigin debian-binary control.tar.gz data.tar.gz
    rm /tmp/combined-contents
    rm control.tar.gz
    rm data.tar.gz
    rm debian-binary
    rm _gpgorigin
  fi
done
echo "Finished GPG signing for deb files"
