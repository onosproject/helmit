// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package bench

import (
	"fmt"
	jobs "github.com/onosproject/helmit/internal/job"
	"github.com/onosproject/helmit/internal/log"
	"os"
)

// The executor is the entrypoint for benchmark images. It takes the input and environment and runs
// the image in the appropriate context according to the arguments.

// Main runs a benchmark
func Main(suites map[string]BenchmarkingSuite) {
	if err := run(suites); err != nil {
		println("Benchmark failed " + err.Error())
		os.Exit(1)
	}
	os.Exit(0)
}

// run runs a benchmark
func run(suites map[string]BenchmarkingSuite) error {
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
	writer := log.NewJSONWriter(os.Stdout)

	job, err := jobs.Bootstrap[Config]()
	if err != nil {
		return err
	}

	executor, err := newExecutor(job.Spec, writer)
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
	job, err := jobs.Bootstrap[WorkerConfig]()
	if err != nil {
		return err
	}

	worker, err := newWorker(job.Spec, suites)
	if err != nil {
		return err
	}
	return worker.run()
}
