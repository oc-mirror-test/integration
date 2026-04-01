#!/bin/bash
#
set -exv

BRANCH=${1:-main}
PR=$3
CLEAN=${2:-false}
REPO=https://github.com/openshift/oc-mirror.git

echo -e "Branch: ${BRANCH} PR: ${PR}"

make build-test-binary-static

if [ "${CLEAN}" == "true" ]; then
    rm -rf oc-mirror
fi

if [ ! -d "oc-mirror" ]; then
    git clone -b ${BRANCH} ${REPO}
fi

cd oc-mirror

if [ "${PR}" == "true" ]; then
    git fetch upstream pull/${PR}/head:${BRANCH}
    git checkout ${BRANCH}
fi

pushd v2 || true

# copy all the relevant directories
rm -rf Makefile
rm -rf containerfile-rhel9-dev
rm -rf scripts/
rm -rf isc/
cp ../Makefile .
cp ../containerfile-rhel9-dev .
cp ../uid_entrypoint.sh .
cp -r ../scripts .
cp -r ../isc .
cp ../registry-config.yaml .
cp -r ../keys .
mkdir -p tests/integration
cp -r ../tests/integration/testdata tests/integration/
mkdir -p bin
cp ../bin/integration.test bin/

make container

echo -e "exit => $?"
