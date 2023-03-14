// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package simulation

import (
	"fmt"
	jobs "github.com/onosproject/helmit/pkg/job"
	"os"
	"path"
)

// The executor is the entrypoint for simulation images. It takes the input and environment and runs
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
				Image:           config.Image,
				ImagePullPolicy: config.ImagePullPolicy,
				Executable:      configExecutable,
				Context:         configContext,
				Values:          config.Values,
				ValueFiles:      configValueFiles,
				Args:            config.Config.Args,
				Env:             config.Env,
				Timeout:         config.Timeout,
			},
			Simulation: config.Simulation,
			Simulators: config.Simulators,
			Duration:   config.Duration,
			Rates:      config.Rates,
			Jitter:     config.Jitter,
			Args:       config.Args,
		},
		Type: simulationJobType,
	}
	return jobs.Run(job)
}

// Main runs a test
func Main(suites map[string]SimulatingSuite) {
	if err := run(suites); err != nil {
		println("Simulator failed " + err.Error())
		os.Exit(1)
	}
	os.Exit(0)
}

// run runs a simulation
func run(suites map[string]SimulatingSuite) error {
	config := &Config{}
	if err := jobs.Bootstrap(config); err != nil {
		return err
	}

	simType := getSimulationType()
	switch simType {
	case simulationTypeCoordinator:
		return runCoordinator(suites, config)
	case simulationTypeWorker:
		return runSimulator(suites, config)
	}
	return nil
}

// runCoordinator runs a test image in the coordinator context
func runCoordinator(suites map[string]SimulatingSuite, config *Config) error {
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

// runSimulator runs a test image in the worker context
func runSimulator(suites map[string]SimulatingSuite, config *Config) error {
	server, err := newWorker(suites, config)
	if err != nil {
		return err
	}
	return server.Run()
}
