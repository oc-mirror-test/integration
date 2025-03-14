#!/bin/bash

set -exv 

# add other variables and ovverrides
LOCAL_REGISTRY=${TEST_REGISTRY:-localhost:5000}


# declare functions
all_happy_path () {
  
  # we can leave the --v2 flag out but this is included incase a full binary was used
  oc-mirror --config isc/isc-happy-path.yaml file://workingdir --v2
  # echo -e "exit => $?"

  # this creates an error regarding graph data - need to investigate it
  # mkdir new-workingdir
  # cp -r workingdir/ new-workingdir/
  oc-mirror --config isc/isc-happy-path.yaml --from file://workingdir docker://${LOCAL_REGISTRY}/integration-tests --v2 --dest-tls-verify=false
  # echo -e "exit => $?"
}

m2d_happy_path () {
  oc-mirror --config isc/isc-happy-path.yaml file://workingdir --v2
  # echo -e "exit => $?"
}

d2m_happy_path () {
  oc-mirror --config isc/isc-happy-path.yaml --from file://workingdir docker://${LOCAL_REGISTRY}/integration-tests --v2 --dest-tls-verify=false
  # echo -e "exit => $?"
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
