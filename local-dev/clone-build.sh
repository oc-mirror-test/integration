#!/bin/bash
#
set -exv 

BRANCH=${1:-main}
PR=$3
CLEAN=${2:-false}
REPO=https://github.com/openshift/oc-mirror.git

echo -e "Branch: ${BRANCH} PR: ${PR}"

if [ "${CLEAN}" == "true" ];
then
  rm -rf oc-mirror
fi

if [ ! -d "oc-mirror" ];
then
  git clone -b ${BRANCH} ${REPO}
fi

cd oc-mirror

if [ "${PR}" == "true" ];
then
  git fetch upstream pull/${PR}/head:${BRANCH} 
  git checkout ${BRANCH}
fi

# cd to v2
cd v2

# copy all the releavnt directories
rm -rf Makefile
rm -rf containerfile-rhel9
rm -rf scripts/
rm -rf isc/
cp ../../Makefile .
cp ../../containerfile-rhel9 .
cp ../../uid_entrypoint.sh .
cp -r ../../scripts .
cp -r ../../isc .

make container

# for the error handling this format is important
echo -e "exit => $?"
