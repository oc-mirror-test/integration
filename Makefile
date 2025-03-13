.PHONY: all test build clean

REGISTRY_BASE ?= quay.io/oc-mirror
IMAGE_NAME ?= integration-tests
IMAGE_VERSION ?= v0.0.1

BTRFS_BUILD_TAG = exclude_graphdriver_devicemapper exclude_graphdriver_btrfs containers_image_openpgp
GO_BUILD_FLAGS = -tags "json1 $(BTRFS_BUILD_TAG)"

all: clean test build

clone:
	$(shell ./scripts/clone.sh $(PR) $(BRANCH))

build: 
	mkdir -p build
	go build -o build ./... 

build-static: 
	mkdir -p build
	go build  -mod=readonly -ldflags="-extldflags=-static" $(GO_BUILD_FLAGS) -o build ./... 


build-dev:
	mkdir -p build
	GOOS=linux go build -ldflags="-s -w" -o build -tags real./...
	chmod 755 build/microservice
	chmod 755 build/uid_entrypoint.sh

verify:
	golangci-lint run -c .golangci.yaml 

test:
	mkdir -p tests/results
	go test -v -short -coverprofile=tests/results/cover.out ./...

test-integration:
	mkdir -p tests/results-integration
	go test -coverprofile=tests/results-integration/cover-additional.out  -race -count=1 ./... -run TestIntegrationAdditional
	go test -coverprofile=tests/results-integration/cover-release.out  -race -count=1 ./... -run TestIntegrationRelease
	go test -coverprofile=tests/results-integration/cover-additional.out  -race -count=1 ./... -run TestIntegrationAdditionalM2M
	go test -coverprofile=tests/results-integration/cover-release.out  -race -count=1 ./... -run TestIntegrationReleaseM2M


cover:
	go tool cover -html=tests/results/cover.out -o tests/results/cover.html

clean:
	rm -rf build/*
	go clean ./...

container:
	podman build -t  ${REGISTRY_BASE}/${IMAGE_NAME}:${IMAGE_VERSION} -f containerfile-rhel9

push:
	podman push --authfile=${HOME}/.docker/config.json ${REGISTRY_BASE}/${IMAGE_NAME}:${IMAGE_VERSION} 
