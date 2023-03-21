// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package test

import (
	"context"
	"fmt"
	"github.com/onosproject/helmit/internal/job"
	"github.com/onosproject/helmit/pkg/helm"
	"github.com/stretchr/testify/suite"
	"k8s.io/client-go/rest"
	"os"
	"testing"
)

// The executor is the entrypoint for benchmark images. It takes the input and environment and runs
// the image in the appropriate context according to the arguments.

// Main runs a benchmark
func Main(suites map[string]TestingSuite) {
	var config Config
	if err := job.Bootstrap(&config); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	if !run(config, suites) {
		os.Exit(1)
	}
	os.Exit(0)
}

// run runs the tests
func run(config Config, suites map[string]TestingSuite) bool {
	var tests []testing.InternalTest
	if len(config.Suites) > 0 {
		for _, name := range config.Suites {
			suite, ok := suites[name]
			if !ok {
				panic(fmt.Sprintf("unknown suite %s", name))
			}
			tests = append(tests, testing.InternalTest{
				Name: name,
				F:    getSuiteFunc(config, suite),
			})
		}
	} else {
		for name := range suites {
			suite := suites[name]
			tests = append(tests, testing.InternalTest{
				Name: name,
				F:    getSuiteFunc(config, suite),
			})
		}
	}

	// Hack to enable verbose testing.
	if config.Verbose {
		os.Args = []string{
			os.Args[0],
			"-test.v",
		}
	}

	testing.Init()
	return testing.RunTests(func(_, _ string) (bool, error) { return true, nil }, tests)
}

func getSuiteFunc(config Config, testingSuite TestingSuite) func(*testing.T) {
	return func(t *testing.T) {
		ctx := context.Background()
		deadline, ok := t.Deadline()
		if ok {
			ctx, cancel := context.WithDeadline(ctx, deadline)
			defer cancel()
			testingSuite.SetContext(ctx)
		} else {
			ctx := context.Background()
			testingSuite.SetContext(ctx)
		}

		testingSuite.SetNamespace(config.Namespace)
		raftConfig, err := rest.InClusterConfig()
		if err == nil {
			t.Fatal(err)
		}
		testingSuite.SetConfig(raftConfig)

		testingSuite.SetHelm(helm.NewClient(helm.Context{
			Namespace:  config.Namespace,
			WorkDir:    config.Context,
			Values:     config.Values,
			ValueFiles: config.ValueFiles,
		}))

		suite.Run(t, testingSuite)

		if config.Parallel {
			t.Parallel()
		}
	}
}
