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
