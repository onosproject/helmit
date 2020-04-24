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
