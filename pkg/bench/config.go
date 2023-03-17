// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package bench

import (
	corev1 "k8s.io/api/core/v1"
	"time"
)

// Config is a benchmark configuration
type Config struct {
	WorkerConfig   `json:"workerConfig"`
	Suite          string            `json:"suite,omitempty"`
	Benchmark      string            `json:"benchmark,omitempty"`
	Workers        int               `json:"workers,omitempty"`
	Parallelism    int               `json:"parallelism,omitempty"`
	Iterations     int               `json:"iterations,omitempty"`
	Duration       *time.Duration    `json:"duration,omitempty"`
	ReportInterval time.Duration     `json:"reportInterval"`
	Args           map[string]string `json:"args,omitempty"`
	NoTeardown     bool              `json:"verbose,omitempty"`
}

// WorkerConfig is a benchmark worker configuration
type WorkerConfig struct {
	Image           string            `json:"workerImage"`
	ImagePullPolicy corev1.PullPolicy `json:"WorkerImagePullPolicy"`
	Env             map[string]string `json:"env"`
}
