.PHONY: all test build clean

REGISTRY_BASE ?= quay.io/oc-mirror
IMAGE_NAME ?= integration-tests
IMAGE_VERSION ?= v0.0.2

BTRFS_BUILD_TAG = exclude_graphdriver_devicemapper exclude_graphdriver_btrfs containers_image_openpgp
GO_BUILD_FLAGS = -tags "json1 $(BTRFS_BUILD_TAG)"

ARTIFACTS_IMAGE_NAME ?= integration-tests-artifacts
ARTIFACTS_IMAGE_VERSION ?= v0.0.2

all: clean test build

clone:
	$(shell ./local-dev/clone-build.sh $(PR) $(BRANCH))

build: 
	mkdir -p bin
	go build -mod=readonly -o bin ./... 

build-static:
	mkdir -p bin
	go build  -mod=readonly -ldflags="-extldflags=-static" $(GO_BUILD_FLAGS) -o bin ./...

build-test-binary:
	mkdir -p bin
	go test -c -mod=readonly $(GO_BUILD_FLAGS) -o bin/integration.test ./tests/integration/

build-test-binary-static:
	mkdir -p bin
	CGO_ENABLED=0 go test -c -mod=readonly -ldflags="-extldflags=-static" $(GO_BUILD_FLAGS) -o bin/integration.test ./tests/integration/

test:
	mkdir -p tests/results
	go test -v -short -coverprofile=tests/results/cover.out ./...

clean:
	rm -rf build/*
	go clean ./...

container:
	podman build -t  ${REGISTRY_BASE}/${IMAGE_NAME}:${IMAGE_VERSION}-dev -f containerfile-rhel9-dev

push:
	podman push --authfile=${HOME}/.docker/config.json ${REGISTRY_BASE}/${IMAGE_NAME}:${IMAGE_VERSION}-dev

container-artifacts:
	podman build -t ${REGISTRY_BASE}/${ARTIFACTS_IMAGE_NAME}:${ARTIFACTS_IMAGE_VERSION} -f containerfile-rhel9-artifacts

push-artifacts:
	podman push --authfile=${HOME}/.docker/config.json ${REGISTRY_BASE}/${ARTIFACTS_IMAGE_NAME}:${ARTIFACTS_IMAGE_VERSION}
