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
	"regexp"
	"runtime/debug"
	"strings"
	"testing"
)

// The executor is the entrypoint for benchmark images. It takes the input and environment and runs
// the image in the appropriate context according to the arguments.

// Main runs a test
func Main(suites []InternalTestSuite) {
	var config Config
	if err := job.Bootstrap(&config); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	var tests []testing.InternalTest
	if len(config.Suites) > 0 {
		for _, match := range config.Suites {
			for _, suite := range suites {
				if ok, _ := regexp.MatchString(match, suite.Name); ok {
					tests = append(tests, testing.InternalTest{
						Name: suite.Name,
						F:    getSuiteFunc(config, suite.Suite),
					})
				}
			}
		}
	} else {
		for _, suite := range suites {
			tests = append(tests, testing.InternalTest{
				Name: suite.Name,
				F:    getSuiteFunc(config, suite.Suite),
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
		for key, value := range config.Args {
			ctx = context.WithValue(ctx, key, value)
		}

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
				if setupSuite, ok := suite.(SetupSuite); ok {
					ctx, cancel := context.WithTimeout(ctx, config.Timeout)
					setupSuite.SetupSuite(ctx)
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

					if tearDownMethod, ok := methodFinder.MethodByName("TearDown" + method.Name); ok {
						ctx, cancel := context.WithTimeout(ctx, config.Timeout)
						defer cancel()
						tearDownMethod.Func.Call([]reflect.Value{reflect.ValueOf(suite), reflect.ValueOf(ctx)})
					}

					if tearDownTest, ok := suite.(TearDownTest); ok {
						ctx, cancel := context.WithTimeout(ctx, config.Timeout)
						defer cancel()
						tearDownTest.TearDownTest(ctx)
					}

					suite.SetT(parentT)
					failOnPanic(t, r)
				}()

				if setupTest, ok := suite.(SetupTest); ok {
					ctx, cancel := context.WithTimeout(ctx, config.Timeout)
					defer cancel()
					setupTest.SetupTest(ctx)
				}

				if setupMethod, ok := methodFinder.MethodByName("Setup" + method.Name); ok {
					ctx, cancel := context.WithTimeout(ctx, config.Timeout)
					defer cancel()
					setupMethod.Func.Call([]reflect.Value{reflect.ValueOf(suite), reflect.ValueOf(ctx)})
				}

				ctx, cancel := context.WithTimeout(ctx, config.Timeout)
				defer cancel()
				method.Func.Call([]reflect.Value{reflect.ValueOf(suite), reflect.ValueOf(ctx)})
			})
		}

		if suiteSetupDone && !config.NoTeardown {
			defer func() {
				if tearDownSuite, ok := suite.(TearDownSuite); ok {
					ctx, cancel := context.WithTimeout(ctx, config.Timeout)
					defer cancel()
					tearDownSuite.TearDownSuite(ctx)
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
