#!/bin/bash

# This is used to verify the ci build
# Acts as a simulation in the ci env
# It is not used at all in ci (PROW, Konflux etc) so ignore it

set -exv 

BRANCH=${1:-main}
CLEAN=${2:-false}
REPO=https://github.com/openshift/oc-mirror.git

echo -e "Branch: ${BRANCH}"

if [ "${CLEAN}" == "true" ];
then
  rm -rf oc-mirror
fi

if [ ! -d "oc-mirror" ];
then
  git clone -b ${BRANCH} ${REPO}
fi

cd oc-mirror

# cd to v2
cd v2

# copy all the relevant directories
rm -rf Makefile
rm -rf containerfile-rhel9-ci
cp ../../Makefile .
cp ../../containerfile-rhel9-ci .
cp ../../uid_entrypoint.sh .

podman build -t quay.io/oc-mirror/integration-tests:v0.0.1 -f containerfile-rhel9-ci

echo -e "exit => $?"
