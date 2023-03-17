// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package bench

import (
	corev1 "k8s.io/api/core/v1"
	"os"
	"strconv"
	"time"
)

type BenchmarkType string

const (
	benchmarkTypeEnv   = "BENCHMARK_TYPE"
	benchmarkWorkerEnv = "BENCHMARK_WORKER"
)

const (
	benchTypeExecutor BenchmarkType = "executor"
	benchTypeWorker   BenchmarkType = "worker"
)

// Config is a benchmark configuration
type Config struct {
	WorkerConfig `json:"workerConfig"`
	Suite        string            `json:"suite,omitempty"`
	Benchmark    string            `json:"benchmark,omitempty"`
	Workers      int               `json:"workers,omitempty"`
	Parallelism  int               `json:"parallelism,omitempty"`
	Iterations   int               `json:"iterations,omitempty"`
	Duration     *time.Duration    `json:"duration,omitempty"`
	Args         map[string]string `json:"args,omitempty"`
	NoTeardown   bool              `json:"verbose,omitempty"`
}

// WorkerConfig is a benchmark worker configuration
type WorkerConfig struct {
	Image           string            `json:"workerImage"`
	ImagePullPolicy corev1.PullPolicy `json:"WorkerImagePullPolicy"`
	Env             map[string]string `json:"env"`
}

// getBenchmarkType returns the current benchmark type
func getBenchmarkType() BenchmarkType {
	context := os.Getenv(benchmarkTypeEnv)
	if context != "" {
		return BenchmarkType(context)
	}
	return benchTypeExecutor
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
