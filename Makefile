# SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
#
# SPDX-License-Identifier: Apache-2.0

export CGO_ENABLED=0
export GO111MODULE=on

.PHONY: build

HELMIT_VERSION := latest

build-tools:=$(shell if [ ! -d "./build/build-tools" ]; then cd build && git clone https://github.com/onosproject/build-tools.git; fi)
include ./build/build-tools/make/onf-common.mk

build: # @HELP build the Go binaries and run all validations (default)
build: build-helmit

build-helmit:
	go build -o build/_output/helmit ./cmd/helmit

build-helmit-tests:
	go build -o build/helmit-tests/_output/bin/onos-tests ./cmd/onos-tests

generate: # @HELP generate k8s client interfaces and implementations
generate:
	go run github.com/onosproject/helmit/cmd/helmit-generate ./build/helmit-generate/generate.yaml ./pkg/kubernetes

test: # @HELP run the unit tests and source code validation
test: linters license build deps
	go test github.com/onosproject/helmit/...

jenkins-test:  # @HELP run the unit tests and source code validation producing a junit style report for Jenkins
jenkins-test: deps license linters
	TEST_PACKAGES=NONE ./build/build-tools/build/jenkins/make-unit

proto: # @HELP build Protobuf/gRPC input types
proto:
	docker run -it -v `pwd`:/go/src/github.com/onosproject/helmit \
		-w /go/src/github.com/onosproject/helmit \
		--entrypoint build/bin/compile_protos.sh \
		onosproject/protoc-go:stable

helmit-runner-docker: # @HELP build helmit-runner Docker image
helmit-runner-docker:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o build/helmit-runner/_output/bin/helmit-runner ./cmd/helmit-runner
	docker build build/helmit-runner -f build/helmit-runner/Dockerfile \
		--build-arg ONOS_BUILD_VERSION=${ONOS_BUILD_VERSION} \
		-t onosproject/helmit-runner:${HELMIT_VERSION}

helmit-tests-docker: # @HELP build helmit tests Docker image
helmit-tests-docker:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o build/helmit-tests/_output/bin/helmit-tests ./cmd/helmit-tests
	docker build build/helmit-tests -f build/helmit-tests/Dockerfile \
		-t onosproject/helmit-tests:${HELMIT_VERSION}

images: # @HELP build all Docker images
images: helmit-runner-docker helmit-tests-docker

kind: # @HELP build Docker images and add them to the currently configured kind cluster
kind: images
	@if [ "`kind get clusters`" = '' ]; then echo "no kind cluster found" && exit 1; fi
	kind load docker-image onosproject/helmit-runner:${HELMIT_VERSION}
	kind load docker-image onosproject/helmit-tests:${HELMIT_VERSION}

all: build images tests

publish: # @HELP publish version on github and dockerhub
	./build/build-tools/publish-version ${VERSION} onosproject/helmit-runner onosproject/helmit-tests

jenkins-publish: # @HELP Jenkins calls this to publish artifacts
	./build/bin/push-images
	./build/build-tools/release-merge-commit

all: t
clean:: # @HELP remove all the build artifacts
	rm -rf ./build/_output ./vendor
