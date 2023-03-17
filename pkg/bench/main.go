// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package bench

import (
	"fmt"
	"github.com/onosproject/helmit/internal/console"
	jobs "github.com/onosproject/helmit/internal/job"
	"os"
)

// The executor is the entrypoint for benchmark images. It takes the input and environment and runs
// the image in the appropriate context according to the arguments.

// Main runs a benchmark
func Main(suites map[string]BenchmarkingSuite) {
	if err := run(suites); err != nil {
		println("B failed " + err.Error())
		os.Exit(1)
	}
	os.Exit(0)
}

// run runs a benchmark
func run(suites map[string]BenchmarkingSuite) error {
	benchType := getBenchmarkType()
	switch benchType {
	case benchTypeExecutor:
		return runExecutor()
	case benchTypeWorker:
		return runWorker(suites)
	}
	return nil
}

// runExecutor runs a test image in the executor context
func runExecutor() error {
	context := console.NewContext(os.Stdout)
	defer context.Close()

	var job jobs.Job[Config]
	err := context.Run("Bootstrapping benchmark executor", func(task *console.Task) error {
		return jobs.Bootstrap[Config](&job)
	})
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
func runWorker(suites map[string]BenchmarkingSuite) error {
	var job jobs.Job[WorkerConfig]
	if err := jobs.Bootstrap[WorkerConfig](&job); err != nil {
		return err
	}

	worker, err := newWorker(job.Spec, suites)
	if err != nil {
		return err
	}
	return worker.run()
}
