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
test: linters license_check build deps
	go test github.com/onosproject/helmit/...

jenkins-test:  # @HELP run the unit tests and source code validation producing a junit style report for Jenkins
jenkins-test: build-tools deps license_check linters
	TEST_PACKAGES=NONE ./../build-tools/build/jenkins/make-unit


coverage: # @HELP generate unit test coverage data
coverage: build deps license_check
	#./build/bin/coveralls-coverage

linters: golang-ci # @HELP examines Go source code and reports coding problems
	golangci-lint run --timeout 5m

build-tools: # @HELP install the ONOS build tools if needed
	@if [ ! -d "../build-tools" ]; then cd .. && git clone https://github.com/onosproject/build-tools.git; fi

jenkins-tools: # @HELP installs tooling needed for Jenkins
	cd .. && go get -u github.com/jstemmer/go-junit-report && go get github.com/t-yuki/gocover-cobertura

golang-ci: # @HELP install golang-ci if not present
	golangci-lint --version || curl -sfL https://install.goreleaser.com/github.com/golangci/golangci-lint.sh | sh -s -- -b `go env GOPATH`/bin v1.36.0

deps: # @HELP ensure that the required dependencies are in place
	go build -v ./...
	bash -c "diff -u <(echo -n) <(git diff go.mod)"
	bash -c "diff -u <(echo -n) <(git diff go.sum)"


license_check: build-tools # @HELP examine and ensure license headers exist
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

publish: # @HELP publish version on github and dockerhub
	./../build-tools/publish-version ${VERSION} onosproject/helmit-runner onosproject/helmit-tests

jenkins-publish: build-tools jenkins-tools # @HELP Jenkins calls this to publish artifacts
	./build/bin/push-images
	../build-tools/release-merge-commit

all: t
clean: # @HELP remove all the build artifacts
	rm -rf ./build/_output ./vendor

help:
	@grep -E '^.*: *# *@HELP' $(MAKEFILE_LIST) \
    | sort \
    | awk ' \
        BEGIN {FS = ": *# *@HELP"}; \
        {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}; \
    '
