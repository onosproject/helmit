// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package test

import (
	"context"
	"errors"
	"fmt"
	grpc_retry "github.com/grpc-ecosystem/go-grpc-middleware/retry"
	api "github.com/onosproject/helmit/api/v1"
	"github.com/onosproject/helmit/internal/console"
	"github.com/onosproject/helmit/internal/job"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"path"
	"time"
)

const managementPort = 5000

// newExecutor returns a new test job
func newExecutor(spec job.Spec) (*testExecutor, error) {
	return &testExecutor{
		spec: spec,
		jobs: job.NewManager[WorkerConfig](job.WorkerType),
	}, nil
}

// testExecutor coordinates workers for suites of tests
type testExecutor struct {
	spec job.Spec
	jobs *job.Manager[WorkerConfig]
}

// Run runs the tests
func (e *testExecutor) run(config Config, context *console.Context) error {
	err := context.Fork("Starting workers", func(context *console.Context) error {
		var joiners []console.Fork
		for i := 0; i < config.Workers; i++ {
			joiners = append(joiners, func(worker int) console.Fork {
				return context.Fork(fmt.Sprintf("Starting worker %d", worker), func(context *console.Context) error {
					return e.createWorker(config, worker, context)
				})
			}(i))
		}
		return console.Join(joiners...)
	}).Join()
	if err != nil {
		return err
	}

	workers := make(map[int]api.TesterClient)
	for i := 0; i < config.Workers; i++ {
		worker, err := grpc.Dial(
			fmt.Sprintf("%s:%d", newWorkerName(e.spec.ID, i), managementPort),
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithUnaryInterceptor(
				grpc_retry.UnaryClientInterceptor(
					grpc_retry.WithCodes(codes.Unavailable),
					grpc_retry.WithBackoff(grpc_retry.BackoffExponential(1*time.Second)),
					grpc_retry.WithMax(10))))
		if err != nil {
			return err
		}
		workers[i] = api.NewTesterClient(worker)
	}

	var allSuites []*api.TestSuite
	for _, worker := range workers {
		response, err := worker.GetTestSuites(e.newContext(config), &api.GetTestSuitesRequest{})
		if err == nil {
			allSuites = response.Suites
			break
		}
	}
	if allSuites == nil {
		return errors.New("failed to load test suites")
	}

	suiteNames := make(map[string]bool)
	for _, suiteName := range config.Suites {
		suiteNames[suiteName] = true
	}

	testNames := make(map[string]bool)
	for _, testName := range config.Tests {
		testNames[testName] = true
	}

	var suites []*api.TestSuite
	for _, suite := range allSuites {
		if len(suiteNames) == 0 || suiteNames[suite.Name] {
			var tests []*api.Test
			for _, test := range suite.Tests {
				if len(testNames) == 0 || testNames[test.Name] {
					tests = append(tests, test)
				}
			}
			suites = append(suites, &api.TestSuite{
				Name:  suite.Name,
				Tests: tests,
			})
		}
	}

	for i, suite := range suites {
		client := workers[i%len(workers)]
		err := context.Fork(fmt.Sprintf("Running suite '%s'", suite.Name), func(context *console.Context) error {
			err := context.Fork("Setting up the suite", func(context *console.Context) error {
				_, err := client.SetupTestSuite(e.newContext(config), &api.SetupTestSuiteRequest{
					Suite: suite.Name,
				})
				return err
			}).Join()
			if err != nil {
				return err
			}

			err = context.Fork("Running tests", func(context *console.Context) error {
				var result error
				var tests []console.Fork
				for _, test := range suite.Tests {
					tests = append(tests, context.Fork(test.Name, func(context *console.Context) error {
						return context.Run(func(status *console.Status) error {
							status.Report("Setting up the test")
							_, err := client.SetupTest(e.newContext(config), &api.SetupTestRequest{
								Suite: suite.Name,
								Test:  test.Name,
							})
							if err != nil {
								return err
							}

							status.Report("Running the test")
							_, err = client.RunTest(e.newContext(config), &api.RunTestRequest{
								Suite: suite.Name,
								Test:  test.Name,
							})
							if err != nil {
								return err
							}

							status.Report("Tearing down the test")
							_, _ = client.TearDownTest(e.newContext(config), &api.TearDownTestRequest{
								Suite: suite.Name,
								Test:  test.Name,
							})
							if err != nil {
								return err
							}
							return nil
						}).Await()
					}))
				}

				if config.Parallel {
					_ = console.Join(tests...)
				} else {
					for _, test := range tests {
						_ = test.Join()
					}
				}
				return result
			}).Join()

			_ = context.Fork("Tearing down the suite", func(context *console.Context) error {
				_, err := client.TearDownTestSuite(e.newContext(config), &api.TearDownTestSuiteRequest{
					Suite: suite.Name,
				})
				return err
			}).Join()
			return err
		}).Join()
		if err != nil {
			return err
		}
	}

	err = context.Fork("Tearing down workers", func(context *console.Context) error {
		var joiners []console.Fork
		for i := 0; i < config.Workers; i++ {
			joiners = append(joiners, func(worker int) console.Fork {
				return context.Fork(fmt.Sprintf("Stopping worker %d", worker), func(context *console.Context) error {
					jobID := newWorkerName(e.spec.ID, worker)
					job := e.newJob(jobID, config)
					return e.jobs.Stop(job, context)
				})
			}(i))
		}
		return console.Join(joiners...)
	}).Join()
	if err != nil {
		return err
	}
	return nil
}

func newWorkerName(jobID string, worker int) string {
	return fmt.Sprintf("%s-worker-%d", jobID, worker)
}

func (e *testExecutor) getWorkerAddress(config Config, worker int) string {
	return fmt.Sprintf("%s:%d", newWorkerName(e.spec.ID, worker), managementPort)
}

// createWorker creates the given worker
func (e *testExecutor) createWorker(config Config, worker int, context *console.Context) error {
	jobID := newWorkerName(e.spec.ID, worker)
	job := e.newJob(jobID, config)
	return e.jobs.Start(job, context)
}

func (e *testExecutor) newJob(id string, config Config) job.Job[WorkerConfig] {
	valueFiles := make(map[string][]string)
	if e.spec.ValueFiles != nil {
		for release, files := range e.spec.ValueFiles {
			releaseFiles := make([]string, 0, len(files))
			for _, file := range files {
				releaseFiles = append(releaseFiles, path.Base(file))
			}
			valueFiles[release] = releaseFiles
		}
	}

	var executable string
	if e.spec.Executable != "" {
		executable = path.Base(e.spec.Executable)
	}

	var context string
	if e.spec.Context != "" {
		context = path.Base(e.spec.Context)
	}

	return job.Job[WorkerConfig]{
		Spec: job.Spec{
			ID:              id,
			Namespace:       e.spec.Namespace,
			ServiceAccount:  e.spec.ServiceAccount,
			Labels:          e.spec.Labels,
			Annotations:     e.spec.Annotations,
			Image:           config.WorkerConfig.Image,
			ImagePullPolicy: config.WorkerConfig.ImagePullPolicy,
			Executable:      executable,
			Context:         context,
			Values:          e.spec.Values,
			ValueFiles:      valueFiles,
			Env:             e.spec.Env,
			Timeout:         e.spec.Timeout,
			NoTeardown:      e.spec.NoTeardown,
			Secrets:         e.spec.Secrets,
			ManagementPort:  managementPort,
		},
	}
}

func (e *testExecutor) newContext(config Config) context.Context {
	ctx := context.Background()
	for key, value := range config.Args {
		ctx = context.WithValue(ctx, key, value)
	}
	return ctx
}
