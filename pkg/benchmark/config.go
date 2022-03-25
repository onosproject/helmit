// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

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
