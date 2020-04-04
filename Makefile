export CGO_ENABLED=0
export GO111MODULE=on

.PHONY: build

HELMIT_VERSION := latest

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
test: license_check build deps 
	go test github.com/onosproject/helmit/pkg/...
	go test github.com/onosproject/helmit/cmd/...

coverage: # @HELP generate unit test coverage data
coverage: build deps license_check
	#./build/bin/coveralls-coverage


linters: # @HELP examines Go source code and reports coding problems
	golangci-lint run

deps: # @HELP ensure that the required dependencies are in place
	go build -v ./...
	bash -c "diff -u <(echo -n) <(git diff go.mod)"
	bash -c "diff -u <(echo -n) <(git diff go.sum)"


license_check: # @HELP examine and ensure license headers exist
	@if [ ! -d "../build-tools" ]; then cd .. && git clone https://github.com/onosproject/build-tools.git; fi
	./../build-tools/licensing/boilerplate.py -v --rootdir=${CURDIR}


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


clean: # @HELP remove all the build artifacts
	rm -rf ./build/_output ./vendor

help:
	@grep -E '^.*: *# *@HELP' $(MAKEFILE_LIST) \
    | sort \
    | awk ' \
        BEGIN {FS = ": *# *@HELP"}; \
        {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}; \
    '
