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

package benchmark

import (
	"github.com/onosproject/helmit/pkg/job"
	"os"
	"strconv"
	"time"
)

type benchmarkType string

const (
	benchmarkTypeEnv = "BENCHMARK_TYPE"
	benchmarkJobType = "benchmark"

	benchmarkJobEnv    = "BENCHMARK_JOB"
	benchmarkWorkerEnv = "BENCHMARK_WORKER"
)

const (
	benchmarkTypeCoordinator benchmarkType = "coordinator"
	benchmarkTypeWorker      benchmarkType = "worker"
)

// Config is a benchmark configuration
type Config struct {
	*job.Config `json:",inline"`
	Suite       string            `json:"suite,omitempty"`
	Benchmark   string            `json:"benchmark,omitempty"`
	Workers     int               `json:"workers,omitempty"`
	Parallelism int               `json:"parallelism,omitempty"`
	Iterations  int               `json:"iterations,omitempty"`
	Duration    *time.Duration    `json:"duration,omitempty"`
	Args        map[string]string `json:"args,omitempty"`
	MaxLatency  *time.Duration    `json:"maxLatency,omitempty"`
	NoTeardown  bool              `json:"verbose,omitempty"`
}

// getBenchmarkType returns the current benchmark type
func getBenchmarkType() benchmarkType {
	context := os.Getenv(benchmarkTypeEnv)
	if context != "" {
		return benchmarkType(context)
	}
	return benchmarkTypeCoordinator
}

// getBenchmarkWorker returns the current benchmark worker number
func getBenchmarkWorker() int {
	worker := os.Getenv(benchmarkWorkerEnv)
	if worker == "" {
		return 0
	}
	i, err := strconv.Atoi(worker)
	if err != nil {
		panic(err)
	}
	return i
}
