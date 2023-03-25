// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package bench

import (
	"time"
)

type Type string

const (
	SetupType    Type = "Setup"
	WorkerType   Type = "Worker"
	TearDownType Type = "TearDown"
)

// Config is a benchmark configuration
type Config struct {
	Type           Type                `json:"type,omitempty"`
	Namespace      string              `json:"namespace,omitempty"`
	Suite          string              `json:"suite,omitempty"`
	Benchmark      string              `json:"benchmark,omitempty"`
	Parallelism    int                 `json:"parallelism,omitempty"`
	ReportInterval time.Duration       `json:"reportInterval,omitempty"`
	Timeout        time.Duration       `json:"timeout,omitempty"`
	Context        string              `json:"context,omitempty"`
	Values         map[string][]string `json:"values,omitempty"`
	ValueFiles     map[string][]string `json:"valueFiles,omitempty"`
	Args           map[string]string   `json:"args,omitempty"`
	NoTeardown     bool                `json:"verbose,omitempty"`
}
