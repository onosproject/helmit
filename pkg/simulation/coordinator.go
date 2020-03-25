// Copyright 2019-present Open Networking Foundation.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package simulation

import (
	"context"
	"fmt"
	"github.com/onosproject/helmet/pkg/job"
	"github.com/onosproject/helmet/pkg/kubernetes/config"
	"github.com/onosproject/helmet/pkg/registry"
	"github.com/onosproject/helmet/pkg/util/async"
	"github.com/onosproject/helmet/pkg/util/logging"
	"google.golang.org/grpc"
	"os"
	"sync"
	"time"
)

// newCoordinator returns a new simulation coordinator
func newCoordinator(config *Config) (*Coordinator, error) {
	return &Coordinator{
		config: config,
	}, nil
}

// Coordinator coordinates workers for suites of simulators
type Coordinator struct {
	config *Config
}

// Run runs the simulations
func (c *Coordinator) Run() error {
	var suites []string
	if c.config.Simulation == "" {
		suites = registry.GetSimulationSuites()
	} else {
		suites = []string{c.config.Simulation}
	}

	workers := make([]*WorkerTask, len(suites))
	for i, suite := range suites {
		jobID := newJobID(c.config.ID, suite)
		env := c.config.Env
		if env == nil {
			env = make(map[string]string)
		}
		env[config.NamespaceEnv] = c.config.ID
		env[simulationTypeEnv] = string(simulationTypeWorker)
		env[simulationWorkerEnv] = fmt.Sprintf("%d", i)
		env[simulationJobEnv] = c.config.ID

		config := &Config{
			Config: &job.Config{
				ID:              jobID,
				Image:           c.config.Config.Image,
				ImagePullPolicy: c.config.Config.ImagePullPolicy,
				Executable:      c.config.Config.Executable,
				Context:         c.config.Config.Context,
				Values:          c.config.Config.Values,
				ValueFiles:      c.config.Config.ValueFiles,
				Env:             env,
				Timeout:         c.config.Config.Timeout,
			},
			Simulation: suite,
			Simulators: c.config.Simulators,
			Rates:      c.config.Rates,
			Jitter:     c.config.Jitter,
			Args:       c.config.Args,
		}
		worker := &WorkerTask{
			runner: job.NewNamespace(jobID),
			config: config,
		}
		workers[i] = worker
	}
	return runWorkers(workers)
}

// runWorkers runs the given test jobs
func runWorkers(tasks []*WorkerTask) error {
	// Start jobs in separate goroutines
	wg := &sync.WaitGroup{}
	errChan := make(chan error, len(tasks))
	codeChan := make(chan int, len(tasks))
	for _, task := range tasks {
		wg.Add(1)
		go func(task *WorkerTask) {
			status, err := task.Run()
			if err != nil {
				errChan <- err
			} else {
				codeChan <- status
			}
			wg.Done()
		}(task)
	}

	// Wait for all jobs to start before proceeding
	go func() {
		wg.Wait()
		close(errChan)
		close(codeChan)
	}()

	// If any job returned an error, return it
	for err := range errChan {
		return err
	}

	// If any job returned a non-zero exit code, exit with it
	for code := range codeChan {
		if code != 0 {
			os.Exit(code)
		}
	}
	return nil
}

// newJobID returns a new unique test job ID
func newJobID(testID, suite string) string {
	return fmt.Sprintf("%s-%s", testID, suite)
}

// WorkerTask manages a single test job for a test worker
type WorkerTask struct {
	runner  *job.Runner
	config  *Config
	workers []SimulatorServiceClient
}

// Run runs the worker job
func (t *WorkerTask) Run() (int, error) {
	// Start the job
	err := t.run()
	if err != nil {
		_ = t.tearDown()
		return 0, err
	}

	// Tear down the cluster if necessary
	_ = t.tearDown()
	return 0, nil
}

// start starts the test job
func (t *WorkerTask) run() error {
	if err := t.runner.CreateNamespace(); err != nil {
		return err
	}
	if err := t.createWorkers(); err != nil {
		return err
	}
	if err := t.setupSimulation(); err != nil {
		return err
	}
	if err := t.setupSimulators(); err != nil {
		return err
	}
	if err := t.runSimulation(); err != nil {
		return err
	}
	return nil
}

func getSimulatorName(worker int) string {
	return fmt.Sprintf("simulator-%d", worker)
}

func (t *WorkerTask) getWorkerAddress(worker int) string {
	return fmt.Sprintf("%s.%s.svc.cluster.local:5000", getSimulatorName(worker), t.config.ID)
}

// createWorkers creates the simulation workers
func (t *WorkerTask) createWorkers() error {
	return async.IterAsync(t.config.Simulators, t.createWorker)
}

// createWorker creates the given worker
func (t *WorkerTask) createWorker(worker int) error {
	return t.runner.StartJob(&job.Job{
		Config:    t.config.Config,
		JobConfig: t.config,
		Type:      simulationJobType,
	})
}

// getSimulators returns the worker clients for the given simulation
func (t *WorkerTask) getSimulators() ([]SimulatorServiceClient, error) {
	if t.workers != nil {
		return t.workers, nil
	}

	workers := make([]SimulatorServiceClient, t.config.Simulators)
	for i := 0; i < t.config.Simulators; i++ {
		worker, err := grpc.Dial(t.getWorkerAddress(i), grpc.WithInsecure())
		if err != nil {
			return nil, err
		}
		workers[i] = NewSimulatorServiceClient(worker)
	}
	t.workers = workers
	return workers, nil
}

// setupSimulation sets up the simulation
func (t *WorkerTask) setupSimulation() error {
	workers, err := t.getSimulators()
	if err != nil {
		return err
	}

	worker := workers[0]
	_, err = worker.SetupSimulation(context.Background(), &SimulationLifecycleRequest{
		Simulation: t.config.Simulation,
		Args:       t.config.Args,
	})
	return err
}

// setupSimulators sets up the simulators
func (t *WorkerTask) setupSimulators() error {
	simulators, err := t.getSimulators()
	if err != nil {
		return err
	}

	wg := &sync.WaitGroup{}
	errCh := make(chan error)
	for i, simulator := range simulators {
		wg.Add(1)
		go func(simulator int, client SimulatorServiceClient) {
			if err := t.setupSimulator(simulator, client); err != nil {
				errCh <- err
			}
			wg.Done()
		}(i, simulator)
	}
	wg.Wait()
	close(errCh)

	for err := range errCh {
		return err
	}
	return nil
}

// setupSimulator sets up the given simulator
func (t *WorkerTask) setupSimulator(simulator int, client SimulatorServiceClient) error {
	step := logging.NewStep(t.config.ID, "Setup simulator %s/%d", t.config.Simulation, simulator)
	step.Start()
	request := &SimulationLifecycleRequest{
		Simulation: t.config.Simulation,
		Args:       t.config.Args,
	}
	_, err := client.SetupSimulator(context.Background(), request)
	if err != nil {
		step.Fail(err)
		return err
	}
	step.Complete()
	return nil
}

// runSimulation runs the given simulations
func (t *WorkerTask) runSimulation() error {
	// Run the simulation for the configured duration
	step := logging.NewStep(t.config.ID, "Run simulation %s", t.config.Simulation)
	step.Start()

	errCh := make(chan error)
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		if err := t.runSimulators(); err != nil {
			errCh <- err
		}
		wg.Done()
	}()

	go func() {
		wg.Wait()
		close(errCh)
	}()

	for err := range errCh {
		step.Fail(err)
		return err
	}
	step.Complete()
	return nil
}

// runSimulators runs the simulation for a goroutine
func (t *WorkerTask) runSimulators() error {
	simulators, err := t.getSimulators()
	if err != nil {
		return err
	}

	wg := &sync.WaitGroup{}
	errCh := make(chan error)
	for i := 0; i < len(simulators); i++ {
		wg.Add(1)
		go func(simulator int, client SimulatorServiceClient) {
			if err := t.runSimulator(simulator, client); err != nil {
				errCh <- err
			}
			wg.Done()
		}(i, simulators[i])
	}
	wg.Wait()
	return nil
}

// runSimulator runs a random simulator
func (t *WorkerTask) runSimulator(simulator int, client SimulatorServiceClient) error {
	step := logging.NewStep(t.config.ID, "Run simulator %s/%d", t.config.Simulation, simulator)
	step.Start()

	if err := t.startSimulator(simulator, client); err != nil {
		step.Fail(err)
		return err
	}

	<-time.After(t.config.Duration)

	if err := t.stopSimulator(simulator, client); err != nil {
		step.Fail(err)
		return err
	}
	return nil
}

// startSimulator starts the given simulator
func (t *WorkerTask) startSimulator(simulator int, client SimulatorServiceClient) error {
	request := &SimulatorRequest{
		Simulation: t.config.Simulation,
	}
	_, err := client.StartSimulator(context.Background(), request)
	return err
}

// stopSimulator stops the given simulator
func (t *WorkerTask) stopSimulator(simulator int, client SimulatorServiceClient) error {
	request := &SimulatorRequest{
		Simulation: t.config.Simulation,
	}
	_, err := client.StopSimulator(context.Background(), request)
	return err
}

// tearDown tears down the job
func (t *WorkerTask) tearDown() error {
	return t.runner.DeleteNamespace()
}
