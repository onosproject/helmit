// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package test

import (
	"fmt"
	"github.com/onosproject/helmit/internal/job"
	"os"
	"testing"
	"time"
)

// Config is a test configuration
type Config struct {
	Namespace  string              `json:"namespace,omitempty"`
	Suites     []string            `json:"suites,omitempty"`
	Tests      []string            `json:"tests,omitempty"`
	Methods    []string            `json:"methods,omitempty"`
	Verbose    bool                `json:"verbose,omitempty"`
	Args       map[string]string   `json:"args,omitempty"`
	Context    string              `json:"context,omitempty"`
	Values     map[string][]string `json:"values,omitempty"`
	ValueFiles map[string][]string `json:"valueFiles,omitempty"`
	Timeout    time.Duration       `json:"timeout,omitempty"`
	NoTeardown bool                `json:"noTeardown,omitempty"`
}

// Main runs a test
func Main(suites []TestingSuite) {
	var config Config
	if err := job.LoadConfig(&config); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	secrets, err := job.LoadSecrets()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	var tests []testing.InternalTest
	for _, suite := range suites {
		name := getSuiteName(suite)
		if isRunnable(name, config.Tests) {
			tests = append(tests, func(suite TestingSuite) testing.InternalTest {
				return testing.InternalTest{
					Name: name,
					F: func(t *testing.T) {
						run(t, suite, config, secrets)
					},
				}
			}(suite))
		}
	}

	// Hack to enable verbose testing.
	os.Args = []string{
		os.Args[0],
		"-test.v",
	}

	testing.Main(func(_, _ string) (bool, error) { return true, nil }, tests, nil, nil)
}
