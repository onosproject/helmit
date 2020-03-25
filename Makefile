export CGO_ENABLED=0
export GO111MODULE=on

.PHONY: build

HELMET_VERSION := latest

build: # @HELP build the Go binaries and run all validations (default)
build: build-helmet

build-helmet:
	go build -o build/_output/helmet ./cmd/helmet

build-helmet-tests:
	go build -o build/helmet-tests/_output/bin/onos-tests ./cmd/onos-tests

generate: # @HELP generate k8s client interfaces and implementations
generate:
	go run github.com/onosproject/helmet/cmd/helmet-generate ./build/helmet-generate/generate.yaml ./pkg/helm/api

test: # @HELP run the unit tests and source code validation
test: license_check build deps linters
	go test github.com/onosproject/helmet/pkg/...
	go test github.com/onosproject/helmet/cmd/...

coverage: # @HELP generate unit test coverage data
coverage: build deps linters license_check
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
	docker run -it -v `pwd`:/go/src/github.com/onosproject/helmet \
		-w /go/src/github.com/onosproject/helmet \
		--entrypoint build/bin/compile_protos.sh \
		onosproject/protoc-go:stable

helmet-runner-docker: # @HELP build helmet-runner Docker image
helmet-runner-docker:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o build/helmet-runner/_output/bin/helmet-runner ./cmd/helmet-runner
	docker build build/helmet-runner -f build/helmet-runner/Dockerfile \
		--build-arg ONOS_BUILD_VERSION=${ONOS_BUILD_VERSION} \
		-t onosproject/helmet-runner:${HELMET_VERSION}

helmet-tests-docker: # @HELP build helmet tests Docker image
helmet-tests-docker:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o build/helmet-tests/_output/bin/helmet-tests ./cmd/helmet-tests
	docker build build/helmet-tests -f build/helmet-tests/Dockerfile \
		-t onosproject/helmet-tests:${HELMET_VERSION}

images: # @HELP build all Docker images
images: helmet-runner-docker helmet-tests-docker

kind: # @HELP build Docker images and add them to the currently configured kind cluster
kind: images
	@if [ "`kind get clusters`" = '' ]; then echo "no kind cluster found" && exit 1; fi
	kind load docker-image onosproject/helmet-runner:${HELMET_VERSION}
	kind load docker-image onosproject/helmet-tests:${HELMET_VERSION}

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
