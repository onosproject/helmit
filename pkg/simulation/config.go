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
	"github.com/onosproject/helmit/pkg/job"
	"os"
	"strconv"
	"time"
)

type simulationType string

const (
	simulationTypeEnv = "SIMULATION_TYPE"
	simulationJobType = "simulation"

	simulationJobEnv    = "SIMULATION_JOB"
	simulationWorkerEnv = "SIMULATION_WORKER"
)

const (
	simulationTypeCoordinator simulationType = "coordinator"
	simulationTypeWorker      simulationType = "worker"
)

// Config is a simulation configuration
type Config struct {
	*job.Config `json:",inline"`
	Simulation  string                   `json:"simulation,omitempty"`
	Simulators  int                      `json:"simulators,omitempty"`
	Duration    time.Duration            `json:"duration,omitempty"`
	Rates       map[string]time.Duration `json:"rates,omitempty"`
	Jitter      map[string]float64       `json:"jitter,omitempty"`
	Args        map[string]string        `json:"args,omitempty"`
}

// getSimulationType returns the current simulation type
func getSimulationType() simulationType {
	context := os.Getenv(simulationTypeEnv)
	if context != "" {
		return simulationType(context)
	}
	return simulationTypeCoordinator
}

// getSimulatorID returns the current simulation worker number
func getSimulatorID() int {
	worker := os.Getenv(simulationWorkerEnv)
	if worker == "" {
		return 0
	}
	i, err := strconv.Atoi(worker)
	if err != nil {
		panic(err)
	}
	return i
}
