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

Once the binaries have been created copy the registry binary to this directory

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
mkdir working-dir
podman run -it --net=host -v /home/lzuccarelli/.docker/:/root/.docker -v ./working-dir/:/artifacts/workingdir a3e3773b0627  bash

# do a mirror to disk
oc-mirror --config isc/isc-happy-path.yaml file://workingdir --v2 --remove-signatures

# do a disk to mirror
# this assumes you have an instance of a registry running on your host
oc-mirror --config isc/isc-happy-path.yaml --from file://workingdir docker://localhost:5000/test --v2 --dest-tls-verify=false
```

To execute a flow use the following command

```bash
# mount the scripts folder for easier debugging
podman run -it --net=host -v /home/lzuccarelli/.docker/:/root/.docker -v ./images/:/artifacts/workingdir -v ./scripts/:/artfifacts/scripts a3e3773b0627  bash
# this will do a a mirror-to-disk and disk-to-mirror
# also assumes you have an external registry (localhost:5000) running
./scripts/flow-controller.sh all_happy_path
```

### Release signature signing and verification

This step has been included and updated in the current artifacts image

This is just for information sake in case there are changes needed to the test-release-index or test-image on quay.io

Create a simple gpg robot account

Execute the following command and create a "fake" account

```
# use something like robot@test.com for an email address
gpg2 -a --full-generate-key 
```

As we have a fixed naming convention for our release image we can now sign it 

Create the relevant directories (if needed)

To be able to push and pull images from quay.io navigate to the web console and got to robot account 

Click on  "oc-mirror+cicd"

Click on Podman Login and execute that command locally (uppend to the command --authfile ~/.docker/robot-quay.json)


```
mkdir ./sigstore
mkdir ./keys

podman image sign  docker://quay.io/oc-mirror/release/test-release-index:v0.0.1 --sign-by robot@test.com --directory ./sigstore --authfile /home/lzuccarelli/.docker/robot-quay.json --log-level=trace
```


Generate the ascii output so that oc-mirror can read in the pk key

```
gpg -a --output ./keys/release-pk.asc --export-secret-key robot@test.com
```

Finally copy the sigstore public key to keys

i.e. as an example

```
cp sigstore/oc-mirror/release/test-release-index\@sha256\=f81792339c8b5934191d18a53b18bc1d584e01a9f37d59c0aa6905b00200aa1b/signature-1 keys/v0.0.1-f81792339c8b5934191d18a53b18bc1d584e01a9f37d59c0aa6905b00200aa1b
```
