// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package benchmark

import (
	"fmt"
	"os"
	"path"

	jobs "github.com/onosproject/helmit/pkg/job"
)

// The executor is the entrypoint for benchmark images. It takes the input and environment and runs
// the image in the appropriate context according to the arguments.

// Run runs the benchmark
func Run(config *Config) error {
	configValueFiles := make(map[string][]string)
	if config.ValueFiles != nil {
		for release, valueFiles := range config.ValueFiles {
			configReleaseFiles := make([]string, 0)
			for _, valueFile := range valueFiles {
				configReleaseFiles = append(configReleaseFiles, path.Base(valueFile))
			}
			configValueFiles[release] = configReleaseFiles
		}
	}

	configExecutable := ""
	if config.Executable != "" {
		configExecutable = path.Base(config.Executable)
	}

	configContext := ""
	if config.Context != "" {
		configContext = path.Base(config.Context)
	}

	job := &jobs.Job{
		Config: config.Config,
		JobConfig: &Config{
			Config: &jobs.Config{
				ID:              config.ID,
				Namespace:       config.Namespace,
				ServiceAccount:  config.ServiceAccount,
				Labels:          config.Labels,
				Annotations:     config.Annotations,
				Image:           config.Image,
				ImagePullPolicy: config.ImagePullPolicy,
				Executable:      configExecutable,
				Context:         configContext,
				Values:          config.Values,
				ValueFiles:      configValueFiles,
				Args:            config.Config.Args,
				Env:             config.Env,
				Timeout:         config.Timeout,
				NoTeardown:      config.NoTeardown,
				Secrets:         config.Config.Secrets,
			},
			Suite:       config.Suite,
			Benchmark:   config.Benchmark,
			Workers:     config.Workers,
			Parallelism: config.Parallelism,
			Iterations:  config.Iterations,
			Duration:    config.Duration,
			Args:        config.Args,
			MaxLatency:  config.MaxLatency,
			NoTeardown:  config.NoTeardown,
		},
		Type: benchmarkJobType,
	}
	return jobs.Run(job)
}

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
	config := &Config{}
	if err := jobs.Bootstrap(config); err != nil {
		return err
	}

	benchType := getBenchmarkType()
	switch benchType {
	case benchmarkTypeCoordinator:
		return runCoordinator(suites, config)
	case benchmarkTypeWorker:
		return runWorker(suites, config)
	}
	return nil
}

// runCoordinator runs a test image in the coordinator context
func runCoordinator(suites map[string]BenchmarkingSuite, config *Config) error {
	coordinator, err := newCoordinator(suites, config)
	if err != nil {
		return err
	}
	status, err := coordinator.Run()
	if err != nil {
		fmt.Println(err)
	}
	os.Exit(status)
	return nil
}

// runWorker runs a test image in the worker context
func runWorker(suites map[string]BenchmarkingSuite, config *Config) error {
	worker, err := newWorker(suites, config)
	if err != nil {
		return err
	}
	return worker.Run()
}
