# Overview

A set of scripts to clone the oc-mirror repo and build a container with  a statically linked binary
of the current branch  (also can handle PR's)

The base container is fairly lightweight as it uses a ubi9-minimal image, with the compiled binary
and with the scripts and isc folders copied into the binary

## Usage

To build the container for local-dev

Execute the following command line

```bash
# this will build from the main brancha
# parameters are 
# $1 branch
# $2 delete oc-mirror directory
# $3 pr (number)
local-dev/clone-build.sh main true 

# to build from a pr
local-dev/clone-build.sh MY-PR-BRANCH true 1073
```

On successful build of the container 

```bash
# clean up images
podman rmi -f $(podman images | awk '{print $1":"$3}' | grep none | cut -d':' -f2)

# list all images 
podman images 

# use the image tag for quay.io/oc-mirror/integration-tests:v0.0.1

# execute the container
# note the mount points 
# - credentials ~/.docker
# - images (for host disk)
podman run -it --net=host -v /home/lzuccarelli/.docker/:/root/.docker -v ./images/:/artifacts/workingdir a3e3773b0627  bash

# do a mirror to disk
oc-mirror --config isc/isc-happy-path.yaml file://workingdir --v2

# do a disk to mirror
# this assumes you have an instance of a registry running on your host
oc-mirror --config isc/isc-happy-path.yaml --from file://workingdir docker://localhost:5000/test --v2 --dest-tls-verify=false
```

