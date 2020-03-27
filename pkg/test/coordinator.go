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

package test

import (
	"context"
	"fmt"
	"github.com/onosproject/helmit/pkg/job"
	"github.com/onosproject/helmit/pkg/kubernetes"
	"github.com/onosproject/helmit/pkg/registry"
	"google.golang.org/grpc"
	"os"
	"strconv"
	"sync"
)

// newCoordinator returns a new test coordinator
func newCoordinator(config *Config) (*Coordinator, error) {
	return &Coordinator{
		config: config,
	}, nil
}

// Coordinator coordinates workers for suites of tests
type Coordinator struct {
	client kubernetes.Client
	config *Config
}

// Run runs the tests
func (c *Coordinator) Run() error {
	for iteration := 1; iteration <= c.config.Iterations || c.config.Iterations < 0; iteration++ {
		suites := c.config.Suites
		if len(suites) == 0 || suites[0] == "" {
			suites = registry.GetTestSuites()
		}
		workers := make([]*WorkerTask, len(suites))
		for i, suite := range suites {
			jobID := newJobID(c.config.ID+"-"+strconv.Itoa(iteration), suite)
			env := c.config.Env
			if env == nil {
				env = make(map[string]string)
			}
			env[testTypeEnv] = string(testTypeWorker)
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
				Suites:     []string{suite},
				Tests:      c.config.Tests,
				Iterations: c.config.Iterations,
			}
			worker := &WorkerTask{
				runner: job.NewNamespace(config.ID),
				config: config,
			}
			workers[i] = worker
		}
		err := runWorkers(workers)
		if err != nil {
			return err
		}
	}
	return nil
}

// runWorkers runs the given test workers
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
	runner *job.Runner
	config *Config
}

// Run runs the worker job
func (t *WorkerTask) Run() (int, error) {
	if err := t.runner.CreateNamespace(); err != nil {
		return 0, err
	}

	job := &job.Job{
		Config:    t.config.Config,
		JobConfig: t.config,
		Type:      testJobType,
	}

	err := t.runner.StartJob(job)
	if err != nil {
		return 0, err
	}

	address := fmt.Sprintf("%s.%s.svc.cluster.local:5000", job.ID, job.ID)
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return 0, err
	}
	client := NewWorkerServiceClient(conn)
	_, err = client.RunTests(context.Background(), &TestRequest{
		Suite: t.config.Suites[0],
		Tests: t.config.Tests,
	})
	if err != nil {
		return 0, err
	}

	status, err := t.runner.WaitForExit(job)
	if err != nil {
		return 0, err
	}
	_ = t.runner.DeleteNamespace()
	return status, err
}
