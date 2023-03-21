// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package test

// Config is a test configuration
type Config struct {
	Namespace  string              `json:"namespace,omitempty"`
	Suites     []string            `json:"suites,omitempty"`
	Tests      []string            `json:"tests,omitempty"`
	Parallel   bool                `json:"parallel,omitempty"`
	Iterations int                 `json:"iterations,omitempty"`
	Verbose    bool                `json:"verbose,omitempty"`
	Args       map[string]string   `json:"args,omitempty"`
	Context    string              `json:"context,omitempty"`
	Values     map[string][]string `json:"values,omitempty"`
	ValueFiles map[string][]string `json:"valueFiles,omitempty"`
	NoTeardown bool                `json:"noTeardown,omitempty"`
}
