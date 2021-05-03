#!/usr/bin/env bash

set -e

VERSION=$1

echo "****************************************"
echo "Creating major version packages for VERSION=${VERSION}"
echo "****************************************"

# Git SHA-1 hash for commits is 40 char long
if [[ ${#VERSION} -eq 40 ]]
then
  echo "Since the version is a commit hash, the major version will be \"latest\"."
  MAJOR_VERSION=latest
else
  IFS='.' read -ra PARTS <<< "$VERSION"
  MAJOR_VERSION=${PARTS[0]}.x
  echo "This is a release version. Will create binaries with \"$MAJOR_VERSION\" suffix."
fi

cd ${BGO_SPACE}/build/dist

for filename in *; do
  newFilename="${filename/$VERSION/$MAJOR_VERSION}"
  if [[ $newFilename != $filename ]]
  then
    cp $filename $newFilename
  fi
done