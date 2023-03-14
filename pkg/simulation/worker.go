// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package simulation

import (
	"context"
	"fmt"
	"github.com/onosproject/helmit/pkg/helm"
	"github.com/onosproject/helmit/pkg/util/logging"
	"google.golang.org/grpc"
	"net"
)

// newWorker returns a new simulator server
func newWorker(suites map[string]SimulatingSuite, config *Config) (*Worker, error) {
	return &Worker{
		suites:      suites,
		config:      config,
		simulations: make(map[string]*S),
	}, nil
}

// Worker listens for simulator requests
type Worker struct {
	suites      map[string]SimulatingSuite
	config      *Config
	simulations map[string]*S
}

// Run runs a simulation
func (w *Worker) Run() error {
	lis, err := net.Listen("tcp", ":5000")
	if err != nil {
		return err
	}
	server := grpc.NewServer()
	RegisterSimulatorServiceServer(server, w)
	return server.Serve(lis)
}

func (w *Worker) getSimulation(name string, args map[string]string) (*S, error) {
	if simulation, ok := w.simulations[name]; ok {
		return simulation, nil
	}
	suite, ok := w.suites[name]
	if !ok {
		return nil, fmt.Errorf("unknown simulation %s", name)
	}

	suite.SetHelm(helm.NewClient(helm.Context{
		Namespace:  w.config.Namespace,
		WorkDir:    w.config.Context,
		Values:     w.config.Values,
		ValueFiles: w.config.ValueFiles,
	}))

	ctx := context.Background()
	for key, value := range args {
		ctx = context.WithValue(ctx, key, value)
	}
	suite.SetContext(ctx)

	simulation := newSimulator(name, getSimulatorID(), suite, w.config)
	w.simulations[name] = simulation
	return simulation, nil
}

// SetupSimulation sets up a simulation suite
func (w *Worker) SetupSimulation(ctx context.Context, request *SimulationLifecycleRequest) (*SimulationLifecycleResponse, error) {
	step := logging.NewStep(fmt.Sprintf("%s/%d", request.Simulation, getSimulatorID()), "SetupSimulation %s", request.Simulation)
	step.Start()

	simulation, err := w.getSimulation(request.Simulation, request.Args)
	if err != nil {
		step.Fail(err)
		return nil, err
	}

	parentCtx := simulation.suite.Context()
	ctx, cancel := context.WithTimeout(parentCtx, w.config.Timeout)
	defer cancel()
	simulation.suite.SetContext(ctx)
	defer simulation.suite.SetContext(parentCtx)

	if err := simulation.setupSimulation(); err != nil {
		step.Fail(err)
		return nil, err
	}
	step.Complete()
	return &SimulationLifecycleResponse{}, nil
}

// TearDownSimulation tears down a simulation suite
func (w *Worker) TearDownSimulation(ctx context.Context, request *SimulationLifecycleRequest) (*SimulationLifecycleResponse, error) {
	step := logging.NewStep(fmt.Sprintf("%s/%d", request.Simulation, getSimulatorID()), "TearDownSimulation %s", request.Simulation)
	step.Start()

	simulation, err := w.getSimulation(request.Simulation, request.Args)
	if err != nil {
		step.Fail(err)
		return nil, err
	}

	parentCtx := simulation.suite.Context()
	ctx, cancel := context.WithTimeout(parentCtx, w.config.Timeout)
	defer cancel()
	simulation.suite.SetContext(ctx)
	defer simulation.suite.SetContext(parentCtx)

	if err := simulation.teardownSimulation(); err != nil {
		step.Fail(err)
		return nil, err
	}
	step.Complete()
	return &SimulationLifecycleResponse{}, nil
}

func (w *Worker) SetupSimulator(ctx context.Context, request *SimulationLifecycleRequest) (*SimulationLifecycleResponse, error) {
	step := logging.NewStep(fmt.Sprintf("%s/%d", request.Simulation, getSimulatorID()), "SetupSimulator %s", request.Simulation)
	step.Start()

	simulation, err := w.getSimulation(request.Simulation, request.Args)
	if err != nil {
		step.Fail(err)
		return nil, err
	}

	parentCtx := simulation.suite.Context()
	ctx, cancel := context.WithTimeout(parentCtx, w.config.Timeout)
	defer cancel()
	simulation.suite.SetContext(ctx)
	defer simulation.suite.SetContext(parentCtx)

	if err := simulation.setupSimulator(); err != nil {
		step.Fail(err)
		return nil, err
	}
	step.Complete()
	return &SimulationLifecycleResponse{}, nil
}

func (w *Worker) TearDownSimulator(ctx context.Context, request *SimulationLifecycleRequest) (*SimulationLifecycleResponse, error) {
	step := logging.NewStep(fmt.Sprintf("%s/%d", request.Simulation, getSimulatorID()), "TearDownSimulator %s", request.Simulation)
	step.Start()

	simulation, err := w.getSimulation(request.Simulation, request.Args)
	if err != nil {
		step.Fail(err)
		return nil, err
	}

	parentCtx := simulation.suite.Context()
	ctx, cancel := context.WithTimeout(parentCtx, w.config.Timeout)
	defer cancel()
	simulation.suite.SetContext(ctx)
	defer simulation.suite.SetContext(parentCtx)

	if err := simulation.teardownSimulator(); err != nil {
		step.Fail(err)
		return nil, err
	}
	step.Complete()
	return &SimulationLifecycleResponse{}, nil
}

func (w *Worker) StartSimulator(ctx context.Context, request *SimulatorRequest) (*SimulatorResponse, error) {
	step := logging.NewStep(fmt.Sprintf("%s/%d", request.Simulation, getSimulatorID()), "StartSimulator %s", request.Simulation)
	step.Start()

	simulation, ok := w.simulations[request.Simulation]
	if !ok {
		err := fmt.Errorf("unknown simulation %s", request.Simulation)
		step.Fail(err)
		return nil, err
	}

	go simulation.start()
	step.Complete()
	return &SimulatorResponse{}, nil
}

func (w *Worker) StopSimulator(ctx context.Context, request *SimulatorRequest) (*SimulatorResponse, error) {
	step := logging.NewStep(fmt.Sprintf("%s/%d", request.Simulation, getSimulatorID()), "StopSimulator %s", request.Simulation)
	step.Start()

	simulation, ok := w.simulations[request.Simulation]
	if !ok {
		err := fmt.Errorf("unknown simulation %s", request.Simulation)
		step.Fail(err)
		return nil, err
	}

	simulation.stop()
	step.Complete()
	return &SimulatorResponse{}, nil
}
