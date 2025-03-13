#!/bin/bash

set -exv

oc-mirror --config isc/isc-happy-path.yaml file://workingdir --v2

echo -e "exit => $?"
