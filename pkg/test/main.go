// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package test

import (
	"fmt"
	"github.com/onosproject/helmit/internal/console"
	jobs "github.com/onosproject/helmit/internal/job"
	"os"
	"testing"
)

// The executor is the entrypoint for benchmark images. It takes the input and environment and runs
// the image in the appropriate context according to the arguments.

// Main runs a benchmark
func Main(suites map[string]TestingSuite) {
	if err := run(suites); err != nil {
		println("Benchmark failed " + err.Error())
		os.Exit(1)
	}
	os.Exit(0)
}

// run runs a benchmark
func run(suites map[string]TestingSuite) error {
	switch jobs.GetType() {
	case jobs.ExecutorType:
		return runExecutor()
	case jobs.WorkerType:
		return runWorker(suites)
	}
	return nil
}

// runExecutor runs a test image in the executor context
func runExecutor() error {
	context := console.NewContext(os.Stdout)
	defer context.Close()

	job, err := jobs.Bootstrap[Config]()
	if err != nil {
		return err
	}

	executor, err := newExecutor(job.Spec)
	if err != nil {
		return err
	}

	if err := executor.run(job.Config, context); err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}

// runWorker runs a test image in the worker context
func runWorker(suites map[string]TestingSuite) error {
	job, err := jobs.Bootstrap[WorkerConfig]()
	if err != nil {
		return err
	}

	tests := []testing.InternalTest{
		{
			Name: "helmit",
			F: func(t *testing.T) {
				worker, err := newWorker(job.Spec, suites, t)
				if err != nil {
					t.Fatal(err)
				}
				if err := worker.run(); err != nil {
					t.Fatal(err)
				}
			},
		},
	}

	testing.Main(func(_, _ string) (bool, error) { return true, nil }, tests, nil, nil)
	return nil
}