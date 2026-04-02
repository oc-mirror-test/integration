# Overview

A set of scripts to clone the oc-mirror repo and build a container with  a statically linked binary
of the current branch  (also can handle PR's)

The base container is fairly lightweight as it uses a ubi9-minimal image, with the compiled binary
and with the scripts and isc folders copied into the binary

## Usage

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
podman run -it --net=host -v /home/${USER}/.docker/:/root/.docker -v ./working-dir/:/artifacts/workingdir a3e3773b0627  bash

# do a mirror to disk
oc-mirror --config isc/isc-happy-path.yaml file://workingdir --v2 --remove-signatures

# do a disk to mirror
# this assumes you have an instance of a registry running on your host
oc-mirror --config isc/isc-happy-path.yaml --from file://workingdir docker://localhost:5000/test --v2 --dest-tls-verify=false
```

To execute a flow use the following command

```bash
# mount the scripts folder for easier debugging
podman run -it --net=host -v /home/${USER}/.docker/:/root/.docker -v ./images/:/artifacts/workingdir -v ./scripts/:/artfifacts/scripts a3e3773b0627  bash
# this will do a a mirror-to-disk and disk-to-mirror
# also assumes you have an external registry (localhost:5000) running
./scripts/flow-controller.sh all_happy_path
```

### Release signature signing and verification

The integration tests require GPG signature verification for the pinned test release image (`quay.io/oc-mirror/release/test-release-index:v0.0.1`). The `keys/` directory is committed to the repo with:

- `release-pk.asc` - GPG public key used by oc-mirror for verification
- A signature file (e.g. `v0.0.1-sha256-<digest>`) - the signed payload for the release image

For running tests locally, no additional setup is needed - the keys in `keys/` should work out of the box.

#### Regenerating keys (only needed when the release image is rebuilt)

If you rebuild the release image (changing its digest), run the following from the repo root:

```bash
./images/release/generate-release-signature.sh
```

This generates a throwaway GPG keypair, signs the release image digest from `images/release/release-payload/index.json`, and writes the public key and signature file to `keys/`. If you update the release image, commit the updated `keys/` directory after running the script.
