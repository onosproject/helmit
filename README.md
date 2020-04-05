# Helmit

### Safety first!

[![Build Status](https://travis-ci.com/onosproject/helmit.svg?branch=master)](https://travis-ci.org/onosproject/helmit)
[![Go Report Card](https://goreportcard.com/badge/github.com/onosproject/helmit)](https://goreportcard.com/report/github.com/onosproject/helmit)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://github.com/gojp/goreportcard/blob/master/LICENSE)
[![GoDoc](https://godoc.org/github.com/onosproject/helmit?status.svg)](https://godoc.org/github.com/onosproject/helmit)

Helmit is a [Golang] framework and tool for end-to-end testing of [Kubernetes] and [Helm] applications.
Helmit supports testing, benchmarking, and simulation inside Kubernetes clusters.

* [Installation](#installation)
* [User Guide](#user-guide)
   1. [Helm API](#helm-api)
   1. [Kubernetes Client](#kubernetes-client)
   1. [Command-Line Tools](#command-line-tools)
   1. [Testing](#testing)
   1. [Benchmarking](#benchmarking)
   1. [Simulation](#simulation)
* [Examples](#examples)
   * [Test Example](./examples/test)
   * [Benchmark Example](./examples/benchmark)
   * [Simulation Example](./examples/simulation)

# Installation

Helmit uses [Go modules](https://github.com/golang/go/wiki/Modules) for dependency management. When installing the
`helmit` command, ensure Go modules are enabled:

```bash
GO111MODULE=on go get github.com/onosproject/helmit/cmd/helmit
```

# User Guide

## Helm API

Helmit provides a Go API for managing Helm charts within a Kubernetes cluster. Tests, benchmarks, and simulations
can use the Helmit Helm API to configure and install charts to test and query resources within releases.

The Helm API is provided by the `github.com/onosproject/helmit/pkg/helm` package:

```go
import "github.com/onosproject/helmit/pkg/helm"

chart := helm.Chart("my-chart")
release := chart.Release("my-release")
```

The Helmit API supports installation of local or remote charts. To install a local chart, use the path 
as the chart name:

```go
helm.Chart("atomix-controller").
	Release("atomix-controller").
	Install(true)
```

To install a remote chart, simply use the chart name:

```go
helm.Chart("kafka").
	Release("kafka").
	Install(true)
```

If the chart repository is not accessible from within the test container, you can optionally specify a repository
URL when creating the chart:

```go
helm.Chart("kafka", "http://storage.googleapis.com/kubernetes-charts-incubator").
	Release("kafka").
	Install(true)
```

The `Install` method installs the chart in the same was as the `helm install` command does. The boolean flags to the
`Install` method indicates whether to block until the chart's resources are ready. 

Release values can be set programmatically using the `Set` receiver:

```go
helm.Chart("kafka", "http://storage.googleapis.com/kubernetes-charts-incubator").
	Release("kafka").
	Set("replicas", 2).
	Set("zookeeper.replicaCount", 3).
	Install(true)
```

Note that values set via command line flags take precedence over programmatically configured values.

## Kubernetes Client

Tests often need to query the resources created by a Helm chart that has been installed. Helmit provides a
custom Kubernetes client designed to query Helm chart resources. The Helmit Kubernetes client looks similar
to the standard [Go client](https://github.com/kubernetes/client-go) but can limit the scope of API calls to
resources transitively owned by a Helm chart release.

To create a Kubernetes client for a release, call `NewForRelease`:

```go
// Create an atomix-controller release
release := helm.Chart("atomix-controller").Release("atomix-controller")

// Create a Kubernetes client scoped for the atomix-controller release
client := kubernetes.NewForReleaseOrDie(release)
```

The release scoped client can be used to list resources created by the release. This can be helpful for e.g.
injecting failures into the cluster during tests:

```go
// Get a list of pods created by the atomix-controller
pods, err := client.CoreV1().Pods().List()
assert.NoError(t, err)

// Get the Atomix controller pod
pod := pods[0]

// Delete the pod
err := pod.Delete()
assert.NoError(t, err)
```

Additionally, Kubernetes objects that create and own other Kubernetes resources -- like `Deployment`, `StatefulSet`, 
`Job`, etc -- provide scoped clients that can be used to query the resources they own as well:

```go
// Get the atomix-controller deployment
deps, err := client.AppsV1().Deployments().List()
assert.NoError(t, err)
assert.Len(t, deps, 1)
dep := deps[0]

// Get the pods created by the controller deployment
pods, err := dep.CoreV1().Pods().List()
assert.NoError(t, err)
assert.Len(t, pods, 1)
pod := pods[0]

// Delete the controller pod
err = pod.Delete()
assert.NoError(t, err)

// Wait a minute for the controller deployment to recover
err = dep.Wait(1 * time.Minute)
assert.NoError(t, err)

// Verify the pod was recovered
pods, err := dep.CoreV1().Pods().List()
assert.NoError(t, err)
assert.Len(t, pods, 1)
assert.NotEqual(t, pod.Name, pods[0].Name)
```

### Code Generation

Like other Kubernetes clients, the Helmit Kubernetes client is generated from a set of templates and Kubernetes
resource metadata using the `helmit-generate` tool.

```bash
go run github.com/onosproject/helmit/cmd/helmit-generate ...
```

Given a [YAML file](./build/helmit-generate/generate.yaml) defining the client's resources, the `helmit-generate` tool 
generates the scoped client code. To generate the base Helmit Kubernetes client, run `make generate`:

```bash
make generate
```

To generate a client with additional resources that are not supported by the base client, define your own
[client configuration](./build/helmit-generate/generate.yaml) and run the tool:

```go
go run github.com/onosproject/helmit/cmd/helmit-generate ./my-client.yaml ./path/to/my/package
```

## Command-Line Tools

The `helmit` command-line tool is used to run tests, benchmarks, and simulations inside a Kubernetes cluster. To
install the `helmit` CLI, use `go get` with Go modules enabled:

```bash
GO111MODULE=on go get github.com/onosproject/helmit/cmd/helmit
```

To use the Helmit CLI, you must have [kubectl](https://kubernetes.io/docs/reference/kubectl/overview/) installed and
configured. Helmit will use the Kubernetes configuration to connect to the cluster to deploy and run tests.

The Helmit CLI consists of only three commands:

* `helmit test` - Runs a [test](#testing) command
* `helmit bench` - Runs a [benchmark](#benchmarking) command
* `helmit sim` - Runs a [simulation](#simulation) command

Each command deploys and runs pods which can deploy Helm charts from within the Kubernetes cluster using the
[Helm API](#helm-api). Each Helmit command supports configuring Helm values in the same way the `helm` command
itself does.

The `helmit` sub-commands support an optional context within which to run tests. When the `--context` flag is set,
the specified context directory will be copied to the Helmit pod running inside Kubernetes and set as the current 
working directory during runs:

```bash
helmit test ./cmd/tests --context ./deploy/charts
```

This allows suites to reference charts by path from within Helmit containers deployed inside Kubernetes:

```go
helm.Chart("./atomix-controller").
	Release("atomix-controller-1").
	Install(true)
```

As with Helm, the `helmit` commands also support values files and flags:

```bash
helmit test ./cmd/tests -f atomix-controller-1=atomix-values.yaml --set atomix-controller-1.replicas=2
```

Because suites may install multiple Helm releases, values files and flags must be prefixed by the *release* name. 
For example, `-f my-release=values.yaml` will add a values file to the release named `my-release`, and
`--set my-release.replicas=3` will set the `replicas` value for the release named `my-release`.

## Testing

Helmit supports testing of [Kubernetes] resources and [Helm] charts using a custom test framework and
command line tool. To test a Kubernetes application, simply [write a Golang test suite](#writing-tests)
and then run the suite using the `helmit test` tool:

```bash
helmit test ./cmd/tests
```

### Writing Tests

Helmit tests are written as suites. When tests are run, each test suite will be deployed and run in its own namespace.
Test suite functions are executed serially.

```go
import "github.com/onosproject/helmit/pkg/test"

type AtomixTestSuite struct {
	test.Suite
}
```

The `SetupTestSuite` interface can be implemented to set up the test namespace prior to running tests:

```go
func (s *AtomixTestSuite) SetupTestSuite() error {
	err := helm.Chart("atomix-controller").
        Release("atomix-controller").
        Set("scope", "Namespace").
        Install(true)
    if err != nil {
        return err
    }

    err = helm.Chart("atomix-database").
        Release("atomix-raft").
        Set("clusters", 3).
        Set("partitions", 10).
        Set("backend.replicas", 3).
        Set("backend.image", "atomix/raft-replica:latest").
        Install(true)
    if err != nil {
        return err
    }
    return nil
}
```

Tests are receivers on the test suite that follow the pattern `Test*`. The standard Golang `testing` library is
used, so all your favorite assertion libraries can be used as well:

```go
import "testing"

func (s *AtomixTestSuite) TestMap(t *testing.T) {
    address, err := s.getController()
    assert.NoError(t, err)

    client, err := atomix.New(address)
    assert.NoError(t, err)

    database, err := client.GetDatabase(context.Background(), "atomix-raft")
    assert.NoError(t, err)

    m, err := database.GetMap(context.Background(), "TestMap")
    assert.NoError(t, err)
}
```

Helmit also supports `TearDownTestSuite` and `TearDownTest` functions for tearing down test suites and tests 
respectively:

```go
func (s *AtomixTestSuite) TearDownTest() error {
	return helm.Chart("atomix-database").
		Release("atomix-database").
		Uninstall()
}
```

### Registering Test Suites

In order to run tests, a main must be provided that registers and names test suites.

```go
import "github.com/onosproject/helmit/pkg/registry"

func init() {
	registry.RegisterTestSuite("atomix", &tests.AtomixTestSuite{})
}
```

Once the tests have been registered, the main should call `test.Main()` to run the tests:

```go

import (
	"github.com/onosproject/helmit/pkg/registry"
	"github.com/onosproject/helmit/pkg/test"
	tests "github.com/onosproject/helmit/test"
)

func main() {
	registry.RegisterTestSuite("atomix", &tests.AtomixTestSuite{})
	test.Main()
}
```

### Running Tests

Once a test suite has been written and registered, running the tests on Kubernetes is simply a matter of running
the `helmit test` command and pointing to the test main:

```bash
helmit test ./cmd/tests
```

When `helmit test` is run with no additional arguments, the test coordinator will run all registered test suites 
in parallel and each within its own namespace. To run a specific test suite, use the `--suite` flag:

```bash
helmit test ./cmd/tests --suite my-tests
```

The `helmit test` command also supports configuring tested Helm charts from the command-line. See the 
[command-line tools](#command-line-tools) documentation for more info.

## Benchmarking

Helmit supports benchmarking of [Kubernetes] resources and [Helm] charts using a custom benchmarking framework and
command line tool. To benchmark a Kubernetes application, simply [write a Golang benchmark suite](#writing-benchmarks)
and then run the suite using the `helmit bench` tool:

```bash
helmit bench ./cmd/benchmarks
```

The benchmark coordinator supports benchmarking a single function on a single node or scaling benchmarks across multiple
containers running inside a Kubernetes cluster.

### Writing Benchmark Suites

To run a benchmark you must first define a benchmark suite. Benchmark suites are Golang structs containing a series 
of receivers to benchmark:

```go
import "github.com/onosproject/helmit/pkg/benchmark"

type AtomixBenchSuite struct {
	benchmark.Suite
}
```

Helmit runs each suite within its own namespace, and each benchmark consists of a `Benchmark*` receivers on the suite.
Prior to running benchmarks, benchmarks suites typically need to set up resources within the Kubernetes namespace.
Benchmarks can implement the following interfaces to manage the namespace:
* `SetupBenchmarkSuite` - Called on a single worker prior to running benchmarks
* `SetupBenchmarkWorker` - Called on each worker pod prior to running benchmarks
* `SetupBenchmark` - Called on each worker pod prior to running each benchmark

Typically, benchmark suites should implement the `SetupBenchmarkSuite` interface to install Helm charts:

```go
func (s *AtomixBenchSuite) SetupBenchmarkSuite(c *benchmark.Context) error {
	err := helm.Chart("atomix-controller").
		Release("atomix-controller").
		Set("scope", "Namespace").
		Install(true)
	if err != nil {
		return err
	}

	err = helm.Chart("atomix-database").
		Release("atomix-raft").
		Set("clusters", 3).
		Set("partitions", 10).
		Set("backend.replicas", 3).
		Set("backend.image", "atomix/raft-replica:latest").
		Install(true)
	if err != nil {
		return err
	}
	return nil
}
```

Benchmarks are written as `Benchmark*` receivers:

```go
func (s *AtomixBenchSuite) BenchmarkMapPut(b *benchmark.Benchmark) error {
    ...
}
```

Each benchmark receiver will be called repeatedly for a configured duration of number of iterations. To generate
randomized benchmark input, the `input` package provides input utilities:

```go
import "github.com/onosproject/helmit/pkg/input"

var keys = input.RandomChoice(input.SetOf(input.RandomString(8), 1000))
var values = input.RandomBytes(128)

func (s *AtomixBenchSuite) BenchmarkMapPut(b *benchmark.Benchmark) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_, err := s.m.Put(ctx, keys.Next().String(), values.Next().Bytes())
	return err
}
```

### Registering Benchmarks

In order to run benchmarks, a main must be provided that registers and names benchmark suites.

```go
import "github.com/onosproject/helmit/pkg/registry"

func init() {
    registry.RegisterBenchmarkSuite("atomix", &tests.AtomixBenchSuite{})
}
```

Once the benchmarks have been registered, the main should call `benchmark.Main()` to run the benchmarks:

```go

import (
	"github.com/onosproject/helmit/pkg/registry"
	"github.com/onosproject/helmit/pkg/benchmark"
	benchmarks "github.com/onosproject/helmit/benchmark"
)

func main() {
    registry.RegisterBenchmarkSuite("atomix", &tests.AtomixBenchSuite{})
    benchmark.Main()
}
```

### Running Benchmarks

Benchmarks are run using the `helmit bench` command. To run a benchmark, run `helmit bench` with the path to
the command in which benchmarks are registered:

```bash
helmit bench ./cmd/benchmarks
```

By default, the `helmit bench` command will run every benchmark suite registered in the provided main.
To run a specific benchmark suite, use the `--suite` flag:

```bash
helmit bench ./cmd/benchmarks --suite atomix
```

To run a specific benchmark function, use the `--benchmark` flag:

```bash
helmit bench ./cmd/benchmarks --suite atomix --benchmark BenchmarkMapPut
```

Benchmarks can either be run for a specific number of iterations:

```bash
helmit bench ./cmd/benchmarks --requests 10000
```

Or for a duration of time:

```bash
helmit bench ./cmd/benchmarks --duration 10m
```

By default, benchmarks are run with a single benchmark goroutine on a single client pod. Benchmarks can be scaled
across many client pods by setting the `--workers` flag:

```go
helmit bench ./cmd/benchmarks --duration 10m --workers 10
```

To scale the number of goroutines within each benchmark worker, set the `--parallel` flag:

```go
helmit bench ./cmd/benchmarks --duration 10m --parallel 10
```

As with all Helmit commands, the `helmit bench` command supports contexts and Helm values and value files:

```bash
helmit bench ./cmd/benchmarks -c . -f kafka=kafka-values.yaml --set kafka.replicas=2 --duration 10m
```

## Simulation

Helmit supports simulation of [Kubernetes] and [Helm] applications using a custom simulation framework and command
line tool. Simulations are collections of operations on a Kubernetes application that are randomized by the Helmit
simulator. To run a simulation, [write a Golang simulation suite](#writing-simulations) and then run the suite using
the `helmit sim` tool:

```bash
helmit sim ./cmd/sims
```

### Writing Simulations

To run a simulation you must first define a simulation suite. Simulation suites are Golang structs containing a series 
of receivers that simulate operations on Kubernetes applications:

```go
import "github.com/onosproject/helmit/pkg/simulation"

type AtomixSimSuite struct {
	simulation.Suite
}
```

Helmit runs each suite within its own namespace, and each simulation consists of a set of simulator functions to run.
Prior to running simulations, simulation suites typically need to set up resources within the Kubernetes namespace.
Simulations can implement the following interfaces to manage the namespace:
* `SetupSimulation` - Called on a single simulation pod prior to running a simulation
* `SetupSimulator` - Called on each simulator pod prior to running a simulation

Typically, simulation suites should implement the `SetupSimulation` interface to install Helm charts:

```go
func (s *AtomixSimSuite) SetupSimulation(c *simulation.Simulator) error {
	err := helm.Chart("atomix-controller").
		Release("atomix-controller").
		Set("scope", "Namespace").
		Install(true)
	if err != nil {
		return err
	}

	err = helm.Chart("atomix-database").
		Release("atomix-raft").
		Set("clusters", 3).
		Set("partitions", 10).
		Set("backend.replicas", 3).
		Set("backend.image", "atomix/raft-replica:latest").
		Install(true)
	if err != nil {
		return err
	}
	return nil
}
```

Simulator functions can be written with any name pattern:

```go
import "github.com/onosproject/helmit/pkg/input"

var keys = input.RandomChoice(input.SetOf(input.RandomString(8), 1000))
var values = input.RandomBytes(128)

func (s *AtomixSimSuite) SimulateMapPut(c *simulation.Simulator) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_, err := s.m.Put(ctx, keys.Next().String(), values.Next().Bytes())
	return err
}
```

Simulations must schedule their simulator functions by implementing the `ScheduleSimulator` interface:

```go
func (s *AtomixSimSuite) ScheduleSimulator(sim *simulation.Simulator) {
	sim.Schedule("get", s.SimulateMapGet, 1*time.Second, 1)
	sim.Schedule("put", s.SimulateMapPut, 5*time.Second, 1)
	sim.Schedule("remove", s.SimulateMapRemove, 30*time.Second, 1)
}
```

When scheduling simulators, the simulation specifies a default rate at which the simulators are executed. Note
that simulator rates can be overridden from the [simulator command line](#running-simulations)

### Registering Simulation Suites

In order to run simulations, a main must be provided that registers and names simulation suites.

```go
import "github.com/onosproject/helmit/pkg/registry"

func init() {
    registry.RegisterSimulationSuite("atomix", &sims.AtomixSimulationSuite{})
}
```

Once the simulations have been registered, the main should call `simulation.Main()` to run the simulations:

```go

import (
	"github.com/onosproject/helmit/pkg/registry"
	"github.com/onosproject/helmit/pkg/simulation"
	simulations "github.com/onosproject/helmit/simulation"
)

func main() {
	registry.RegisterSimulationSuite("atomix", &sims.AtomixSimulationSuite{})
	simulation.Main()
}
```

### Running Simulations

Simulations are run using the `helmit sim` command. To run a simulation, run `helmit sim` with the path to
the command in which simulations are registered:

```bash
helmit sim ./cmd/simulations
```

By default, the `helmit sim` command will run every simulation registered in the provided main.
To run a specific simulation, use the `--simulation` flag:

```bash
helmit sim ./cmd/simulations --suite atomix
```

Simulations can either be run for a configurable duration of time:

```bash
helmit sim ./cmd/simulations --duration 10m
```

By default, simulations are run on a single client pod. Simulations can be scaled across many client pods by 
setting the `--simulators` flag:

```go
helmit sim ./cmd/simulations --duration 10m --simulators 10
```

As with all Helmit commands, the `helmit sim` command supports contexts and Helm values and value files:

```bash
helmit sim ./cmd/simulations -c . -f kafka=kafka-values.yaml --set kafka.replicas=2 --duration 10m
```

# Examples

* [Test Example](./examples/test)
* [Benchmark Example](./examples/benchmark)
* [Simulation Example](./examples/simulation)

[Golang]: https://golang.org/
[Helm]: https://helm.sh
[Kubernetes]: https://kubernetes.io
[ONOS]: https://onosproject.org
[Atomix]: https://atomix.io
