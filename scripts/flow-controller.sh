#!/bin/bash

set -exv 

# add other variables and overrides
REGISTRY=${TEST_REGISTRY:-localhost:5000}
FLAGS=""

if [ "${REGISTRY}" == "localhost:5000" ];
then
  FLAGS="${FLAGS} --dest-tls-verify=false"
fi


# declare functions
all_happy_path () {

  # start the registry in the background
  registry serve registry-config.yaml > /dev/null 2>&1 &
  PID=$!

  # mirror-to-disk
  oc-mirror --config isc/isc-happy-path.yaml file://workingdir --v2
  # echo -e "exit => $?"

  # this creates an error regarding graph data - need to investigate it
  # mkdir new-workingdir
  # cp -r workingdir/ new-workingdir/
  # disk-to-mirror
  oc-mirror --config isc/isc-happy-path.yaml --from file://workingdir docker://${REGISTRY} --v2 ${FLAGS}
  # echo -e "exit => $?"

  # shut the registry down
  kill -9 ${PID}
}

m2d_happy_path () {
  # mirror-to-disk
  oc-mirror --config isc/isc-happy-path.yaml file://workingdir --v2
  # echo -e "exit => $?"
}

d2m_happy_path () {
  # start the registry in the background
  registry serve registry-config.yaml > /dev/null 2>&1 &
  PID=$!

  oc-mirror --config isc/isc-happy-path.yaml --from file://workingdir docker://${REGISTRY} --v2 ${FLAGS}
  # echo -e "exit => $?"
  
  # shut the registry down
  kill -9 ${PID}
}

# enable this when we dont want to exit
# usefull for when we are expecting an error
# err_message() {
#    echo "error occured on line $1"
# }

# trap 'err_message $LINENO' ERR

echo -e "$@"

# main entry point
# add other flows that will include 
# happy path and forced errors
case $1 in

  "all_happy_path")
    all_happy_path
    ;;

  "m2d_happy_path")
    m2d_happy_path
    ;;

  "d2m_happy_path")
    d2m_happy_path
    ;;
 
  *)
    echo "not implemented"
    ;;

esac
