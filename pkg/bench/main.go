// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package bench

import (
	"fmt"
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
		return runExecutor(suites)
	case benchTypeWorker:
		return runWorker(suites)
	}
	return nil
}

// runExecutor runs a test image in the executor context
func runExecutor(suites map[string]BenchmarkingSuite) error {
	status := console.NewReporter(os.Stdout)
	status.Start()
	defer status.Stop()

	job, err := jobs.Bootstrap[Config]()
	if err != nil {
		return err
	}

	executor, err := newExecutor(job.Spec, suites, console.NewContextManager(status))
	if err != nil {
		return err
	}

	if err := executor.run(job.Config); err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}

// runWorker runs a test image in the worker context
func runWorker(suites map[string]BenchmarkingSuite) error {
	job, err := jobs.Bootstrap[Config]()
	if err != nil {
		return err
	}

	worker, err := newWorker(job.Spec, suites)
	if err != nil {
		return err
	}
	return worker.Run()
}
