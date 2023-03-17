// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package test

import (
	corev1 "k8s.io/api/core/v1"
)

// Config is a benchmark configuration
type Config struct {
	WorkerConfig `json:"workerConfig"`
	Suites       []string          `json:"suites,omitempty"`
	Tests        []string          `json:"tests,omitempty"`
	Workers      int               `json:"workers,omitempty"`
	Parallel     bool              `json:"parallel,omitempty"`
	Iterations   int               `json:"iterations,omitempty"`
	Verbose      bool              `json:"verbose,omitempty"`
	Args         map[string]string `json:"args,omitempty"`
	NoTeardown   bool              `json:"noteardown,omitempty"`
}

// WorkerConfig is a benchmark worker configuration
type WorkerConfig struct {
	Image           string            `json:"workerImage"`
	ImagePullPolicy corev1.PullPolicy `json:"WorkerImagePullPolicy"`
	Env             map[string]string `json:"env"`
}
