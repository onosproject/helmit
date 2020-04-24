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


## Examples

The links to examples for testing, benchmarking, simulation are listed as follows:

   * [Test Example](https://github.com/onosproject/helmit/tree/master/examples/test)
   * [Benchmark Example](https://github.com/onosproject/helmit/tree/master/examples/benchmark)
   * [Simulation Example](https://github.com/onosproject/helmit/tree/master/examples/simulation)
   * [Helm Charts Example](https://github.com/onosproject/helmit/tree/master/examples/charts)