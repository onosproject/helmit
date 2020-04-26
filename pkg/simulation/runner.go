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
func Main() {
	if err := run(); err != nil {
		println("Simulator failed " + err.Error())
		os.Exit(1)
	}
	os.Exit(0)
}

// run runs a simulation
func run() error {
	config := &Config{}
	if err := jobs.Bootstrap(config); err != nil {
		return err
	}

	simType := getSimulationType()
	switch simType {
	case simulationTypeCoordinator:
		return runCoordinator(config)
	case simulationTypeWorker:
		return runSimulator(config)
	}
	return nil
}

// runCoordinator runs a test image in the coordinator context
func runCoordinator(config *Config) error {
	coordinator, err := newCoordinator(config)
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
func runSimulator(config *Config) error {
	server, err := newSimulatorServer(config)
	if err != nil {
		return err
	}
	return server.Run()
}
