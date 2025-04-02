# Overview

A set of scripts to clone the oc-mirror repo and build a container with  a statically linked binary
of the current branch  (also can handle PR's)

The base container is fairly lightweight as it uses a ubi9-minimal image, with the compiled binary
and with the scripts and isc folders copied into the binary

## Usage

Befor building the artifacts container we need to include the distribution/distribution (registry) binary in the container

To do this, first clone the distribution project 

```bash

git clone git@github.com:distribution/distribution.git

cd  distribution

DISABLE_CGO=1 CGO_ENABLED=0  make binaries
```

Once the binaries have been created copoy the registry binary to this directory

```bash

cd oc-mirror-test/integration

cp <path-to-distribution>/distrubtion/bin/registry .

```

### Build and push the artifacts container

```bash

# build
podman build -t quay.io/oc-mirror/integration-tests-artifacts:v0.0.1 -f containerfile-rhel9-artifacts

# push 
podman push  quay.io/oc-mirror/integration-tests-artifacts:v0.0.1
```

The following step are for local dev testing and can be ignored 

### Build the local-dev container for testing

To build the container for local-dev

Execute the following command line

```bash
# this will build from the main branch
# parameters are 
#  $1 branch
#  $2 delete oc-mirror directory
#  $3 pr (number)
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

# use the image tag for quay.io/oc-mirror/integration-tests:v0.0.1-dev
# or just use the full name i.e quay.io/oc-mirror/integrations-tests-artifacts:v0.0.1-dev

# execute the container
# note the mount points 
# - credentials ~/.docker
# - images (for host disk)
mkdir images
podman run -it --net=host -v /home/lzuccarelli/.docker/:/root/.docker -v ./images/:/artifacts/workingdir a3e3773b0627  bash

# do a mirror to disk
oc-mirror --config isc/isc-happy-path.yaml file://workingdir --v2

# do a disk to mirror
# this assumes you have an instance of a registry running on your host
oc-mirror --config isc/isc-happy-path.yaml --from file://workingdir docker://localhost:5000/test --v2 --dest-tls-verify=false
```

To execute a flow use the following command

```bash
# mount the scripts folder for easier debugging
podman run -it --net=host -v /home/lzuccarelli/.docker/:/root/.docker -v ./images/:/artifacts/workingdir -v ./scripts/:artfifacts/scripts a3e3773b0627  bash
# this will do a a mirror-to-disk and disk-to-mirror
# also assumes you have an external registry (localhost:5000) running
./scripts/flow-controller.sh all_happy_path
```
