// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package test

import (
	"context"
	"fmt"
	"github.com/onosproject/helmit/internal/job"
	"github.com/onosproject/helmit/pkg/helm"
	"k8s.io/client-go/rest"
	"os"
	"reflect"
	"runtime/debug"
	"strings"
	"testing"
)

// The executor is the entrypoint for benchmark images. It takes the input and environment and runs
// the image in the appropriate context according to the arguments.

// Main runs a test
func Main(suites map[string]TestingSuite) {
	var config Config
	if err := job.Bootstrap(&config); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	var tests []testing.InternalTest
	if len(config.Suites) > 0 {
		for _, name := range config.Suites {
			suite, ok := suites[name]
			if !ok {
				continue
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

	testing.Main(func(_, _ string) (bool, error) { return true, nil }, tests, nil, nil)
}

func getSuiteFunc(config Config, suite TestingSuite) func(*testing.T) {
	return func(t *testing.T) {
		defer recoverAndFailOnPanic(t)

		ctx := context.Background()

		suite.SetT(t)
		suite.SetNamespace(config.Namespace)
		raftConfig, err := rest.InClusterConfig()
		if err != nil {
			t.Fatal(err)
		}
		suite.SetConfig(raftConfig)

		suite.SetHelm(helm.NewClient(helm.Context{
			Namespace:  config.Namespace,
			WorkDir:    config.Context,
			Values:     config.Values,
			ValueFiles: config.ValueFiles,
		}))

		var suiteSetupDone bool
		methodFinder := reflect.TypeOf(suite)
		testNames := config.Tests
		if len(testNames) == 0 {
			for i := 0; i < methodFinder.NumMethod(); i++ {
				method := methodFinder.Method(i)
				if strings.HasPrefix(method.Name, "Test") {
					testNames = append(testNames, method.Name)
				}
			}
		}

		for _, name := range testNames {
			method, ok := methodFinder.MethodByName(name)
			if !ok {
				continue
			}

			if !suiteSetupDone {
				if setupAllSuite, ok := suite.(SetupSuite); ok {
					ctx, cancel := context.WithTimeout(ctx, config.Timeout)
					setupAllSuite.SetupSuite(ctx)
					cancel()
				}
				suiteSetupDone = true
			}

			t.Run(name, func(t *testing.T) {
				parentT := suite.T()
				suite.SetT(t)
				defer recoverAndFailOnPanic(t)
				defer func() {
					r := recover()

					if tearDownTestSuite, ok := suite.(TearDownTest); ok {
						ctx, cancel := context.WithTimeout(ctx, config.Timeout)
						defer cancel()
						tearDownTestSuite.TearDownTest(ctx)
					}

					suite.SetT(parentT)
					failOnPanic(t, r)
				}()

				if setupTestSuite, ok := suite.(SetupTest); ok {
					ctx, cancel := context.WithTimeout(ctx, config.Timeout)
					defer cancel()
					setupTestSuite.SetupTest(ctx)
				}

				ctx, cancel := context.WithTimeout(ctx, config.Timeout)
				defer cancel()
				method.Func.Call([]reflect.Value{reflect.ValueOf(suite), reflect.ValueOf(ctx)})
			})
		}

		if suiteSetupDone {
			defer func() {
				if tearDownAllSuite, ok := suite.(TearDownSuite); ok {
					ctx, cancel := context.WithTimeout(ctx, config.Timeout)
					defer cancel()
					tearDownAllSuite.TearDownSuite(ctx)
				}
			}()
		}

		if config.Parallel {
			t.Parallel()
		}
	}
}

func recoverAndFailOnPanic(t *testing.T) {
	r := recover()
	failOnPanic(t, r)
}

func failOnPanic(t *testing.T, r interface{}) {
	if r != nil {
		t.Errorf("test panicked: %v\n%s", r, debug.Stack())
		t.FailNow()
	}
}
