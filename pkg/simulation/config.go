// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

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
