// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package test

import (
	"github.com/onosproject/helmit/pkg/job"
	"os"
)

type testType string

const (
	testTypeCoordinator testType = "coordinator"
	testTypeWorker      testType = "worker"
)

const (
	testTypeEnv = "TEST_TYPE"
	testJobType = "test"
)

// Config is a test configuration
type Config struct {
	*job.Config `json:",inline"`
	Suites      []string          `json:"suites,omitempty"`
	Tests       []string          `json:"tests,omitempty"`
	Iterations  int               `json:"iterations,omitempty"`
	Verbose     bool              `json:"verbose,omitempty"`
	NoTeardown  bool              `json:"noteardown,omitempty"`
	Args        map[string]string `json:"args,omitempty"`
}

// getTestContext returns the current test context
func getTestType() testType {
	context := os.Getenv(testTypeEnv)
	if context != "" {
		return testType(context)
	}
	return testTypeCoordinator
}
