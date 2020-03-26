# Helmet

### Safety first!

Helmet is a [Golang] framework and tool for end-to-end testing of [Kubernetes] and [Helm] applications.
Helmet supports testing, benchmarking, and simulation inside Kubernetes clusters.

* [Testing](#testing)
* [Benchmarking](#benchmarking)
* [Simulation](#simulation)

## Installation

Helmet uses [Go modules](https://github.com/golang/go/wiki/Modules) for dependency management. When installing the
`helmet` command, ensure Go modules are enabled:

```bash
GO111MODULE=on go get github.com/onosproject/helmet/cmd/helmet
```

## Testing

Helmet supports testing of [Kubernetes] resources and [Helm] charts using a custom test framework and
command line tool. To test a Kubernetes application, simply [write a Golang test suite](#writing-tests)
and then run the suite using the `helmet test` tool:

```bash
helmet test ./cmd/tests
```

### Writing Tests

Helmet tests are written as suites. When tests are run, each test suite will be deployed and run in its own namespace.
Test suite functions are executed serially.

```go
import "github.com/onosproject/helmet/pkg/test"

type MyTestSuite struct {
	test.Suite
}
```

The `SetupTestSuite` interface can be implemented to set up the test namespace prior to running tests:

```go
func (s *MyTestSuite) SetupTestSuite() error {
	
}
```

Typically, suite or test setup functions are used to deploy Kubernetes resources within the test namespace, whether
using the Kubernetes Golang client API or Helm. For Helm deployments, Helmet provides an API for deploying Helm charts.
The Helm API can be used to install and uninstall local or remote Helm charts. To install a local chart, use
the path as the chart name:

```go
helm.Chart("/opt/charts/atomix-controller").
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

Finally, the Helm API supports overriding chart values via the `Set` method:

```go
helm.Chart("kafka", "http://storage.googleapis.com/kubernetes-charts-incubator").
	Release("kafka").
	Set("replicas", 2).
	Set("zookeeper.replicaCount", 3).
	Install(true)
```

Note that the test tool supports value files and overrides via command line flags. When values are specified via
the test CLI, values in code (like above) can be overwritten.

Tests are receivers on the test suite that follow the pattern `Test*`. The standard Golang `testing` library is
used, so all your favorite assertion libraries can be used as well:

```go
import "testing"

func (s *MyTestSuite) TestFoo(t *testing.T) error {
	// Do some assertions
	return nil
}
```

Helmet also supports `TearDownTestSuite` and `TearDownTest` functions for tearing down test suites and tests 
respectively:

```go
func (s *MyTestSuite) TearDownTest() error {
	return helm.Chart("kafka").
		Release("kafka").
		Uninstall()
}
```

### Registering Test Suites

In order to run tests, a main must be provided that registers and names test suites.

```go
import "github.com/onosproject/helmet/pkg/registry"

func init() {
    registry.RegisterTestSuite("my-tests", &tests.MyTestSuite{})
}
```

Once the tests have been registered, the main should call `test.Main()` to run the tests:

```go

import (
	"github.com/onosproject/helmet/pkg/registry"
	"github.com/onosproject/helmet/pkg/test"
	tests "github.com/onosproject/helmet/test"
)

func main() {
    registry.RegisterTestSuite("my-tests", &tests.MyTestSuite{})
    test.Main()
}
```

### Running Tests

Once a test suite has been written and registered, running the tests on Kubernetes is simply a matter of running
the `helmet test` command and pointing to the test main:

```bash
helmet test ./cmd/tests
```

When `helmet test` is run with no additional arguments, the test coordinator will run all registered test suites 
in parallel and each within its own namespace. To run a specific test suite, use the `--suite` flag:

```bash
helmet test ./cmd/tests --suite my-tests
```

Sometimes test suites need to operate on Helm charts that are not available via a remote repository. The `helmet test`
command supports an optional context within which to run tests. When the `--context` flag is set, the specified
context directory will be copied to the test image running inside Kubernetes and set as the current working directory
during test runs:

```bash
helmet test ./cmd/tests --context ./deploy/charts
```

This allows tests to reference charts by path from within test containers deployed inside Kubernetes:

```go
helm.Chart("./atomix-controller").
	Release("atomix-controller-1").
	Install(true)
```

As with Helm, the `helmet test` command also supports values files and flags:

```bash
helmet test ./cmd/tests -f atomix-controller-1=atomix-values.yaml --set atomix-controller-1.replicas=2
```

Because tests may install multiple Helm releases, values files and flags must be prefixed by the *release* name. 
For example, `-f my-release=values.yaml` will add a values file to the release named `my-release`, and
`--set my-release.replicas=3` will set the `replicas` value for the release named `my-release`.

## Benchmarking

Helmet supports benchmarking of [Kubernetes] resources and [Helm] charts using a custom benchmarking framework and
command line tool. To benchmark a Kubernetes application, simply [write a Golang benchmark suite](#writing-benchmarks)
and then run the suite using the `helmet bench` tool:

```bash
helmet bench ./cmd/benchmarks
```

### Writing Benchmark Suites

### Registering Benchmarks

In order to run benchmarks, a main must be provided that registers and names benchmark suites.

```go
import "github.com/onosproject/helmet/pkg/registry"

func init() {
    registry.RegisterBenchmarkSuite("my-bench", &tests.MyBenchSuite{})
}
```

Once the benchmarks have been registered, the main should call `benchmark.Main()` to run the benchmarks:

```go

import (
	"github.com/onosproject/helmet/pkg/registry"
	"github.com/onosproject/helmet/pkg/benchmark"
	benchmarks "github.com/onosproject/helmet/benchmark"
)

func main() {
    registry.RegisterBenchmarkSuite("my-bench", &tests.MyBenchSuite{})
    benchmark.Main()
}
```

### Running Benchmarks

## Simulation

Helmet supports simulation of [Kubernetes] and [Helm] applications using a custom simulation framework and command
line tool. Simulations are collections of operations on a Kubernetes application that are randomized by the Helmet
simulator. To run a simulation, [write a Golang simulation suite](#writing-simulations) and then run the suite using
the `helmet sim` tool:

```bash
helmet sim ./cmd/sims
```

### Writing Simulations

### Registering Simulation Suites

In order to run simulations, a main must be provided that registers and names simulation suites.

```go
import "github.com/onosproject/helmet/pkg/registry"

func init() {
    registry.RegisterSimulationSuite("my-bench", &tests.MySimSuite{})
}
```

Once the simulations have been registered, the main should call `simulation.Main()` to run the simulations:

```go

import (
	"github.com/onosproject/helmet/pkg/registry"
	"github.com/onosproject/helmet/pkg/simulation"
	simulations "github.com/onosproject/helmet/simulation"
)

func main() {
    registry.RegisterSimulationSuite("my-sim", &tests.MySimSuite{})
    simulation.Main()
}
```

### Running Simulations

[Golang]: https://golang.org/
[Helm]: https://helm.sh
[Kubernetes]: https://kubernetes.io
[ONOS]: https://onosproject.org
[Atomix]: https://atomix.io
