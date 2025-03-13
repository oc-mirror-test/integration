#!/bin/bash

set -exv

oc-mirror --config isc/isc-happy-path.yaml --from file://workingdir docker://localhost:5000/test --v2 --dest-tls-verify=false

echo -e "exit => $?"
