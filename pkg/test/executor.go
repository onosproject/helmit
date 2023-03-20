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
	"github.com/onosproject/helmit/internal/job"
	"github.com/onosproject/helmit/internal/log"
	"github.com/onosproject/helmit/internal/task"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"io"
	"path"
	"time"
)

const managementPort = 5000

// newExecutor returns a new test job
func newExecutor(spec job.Spec, writer log.Writer) (*testExecutor, error) {
	return &testExecutor{
		spec:   spec,
		jobs:   job.NewManager[WorkerConfig](job.WorkerType),
		writer: writer,
	}, nil
}

// testExecutor coordinates workers for suites of tests
type testExecutor struct {
	spec   job.Spec
	jobs   *job.Manager[WorkerConfig]
	writer log.Writer
}

// Run runs the tests
func (e *testExecutor) run(config Config) error {
	jobs := make(map[int]job.Job[WorkerConfig])
	for worker := 0; worker < config.Workers; worker++ {
		jobID := newWorkerName(e.spec.ID, worker)
		jobs[worker] = e.newJob(jobID, config)
	}

	err := task.New(e.writer, "Setup test workers").Run(func(context task.Context) error {
		var futures []task.Future
		for i := 0; i < config.Workers; i++ {
			futures = append(futures, func(worker int) task.Future {
				return context.NewTask("Setup test worker %d", worker).Start(func(context task.Context) error {
					context.Status().Setf("Starting worker %d", worker)
					if err := e.jobs.Start(jobs[worker], context.Status()); err != nil {
						return err
					}
					context.Status().Setf("Running worker %d", worker)
					if err := e.jobs.Run(jobs[worker], context.Status()); err != nil {
						return err
					}
					return nil
				})
			}(i))
		}
		return task.Await(futures...)
	})
	if err != nil {
		return err
	}

	clients := make(map[int]api.TesterClient)
	for worker := 0; worker < config.Workers; worker++ {
		client, err := grpc.Dial(
			fmt.Sprintf("%s:%d", newWorkerName(e.spec.ID, worker), managementPort),
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithUnaryInterceptor(
				grpc_retry.UnaryClientInterceptor(
					grpc_retry.WithCodes(codes.Unavailable),
					grpc_retry.WithBackoff(grpc_retry.BackoffExponential(1*time.Second)),
					grpc_retry.WithMax(10))))
		if err != nil {
			return err
		}
		clients[worker] = api.NewTesterClient(client)
	}

	streams := make(map[int]io.ReadCloser)
	for worker := 0; worker < config.Workers; worker++ {
		stream, err := e.jobs.Stream(jobs[worker])
		if err != nil {
			return err
		}
		streams[worker] = stream
	}

	var allSuites []*api.TestSuite
	for _, worker := range clients {
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

	var tasks []task.Task
	for _, suite := range suites {
		tasks = append(tasks, task.New(e.writer, "Run suite '%s'", suite.Name))
	}

	var testErr error
	for i, suite := range suites {
		client := clients[i%len(clients)]
		stream := streams[i%len(streams)]
		err := tasks[i].Run(func(c task.Context) error {
			c.Status().Setf("Setting up '%s'", suite.Name)
			_, err := client.SetupTestSuite(e.newContext(config), &api.SetupTestSuiteRequest{
				Suite: suite.Name,
			})
			if err != nil {
				return err
			}

			var tests []task.Task
			for _, test := range suite.Tests {
				tests = append(tests, c.NewTask(test.Name))
			}

			var newTestRunner = func(test *api.Test) func(task.Context) error {
				return func(c task.Context) error {
					c.Status().Setf("Setting up '%s'", test.Name)
					c.Writer()

					if config.Verbose {
						ctx, cancel := context.WithCancel(context.Background())
						defer cancel()
						go copyContext(ctx, c.Writer(), stream)
					}

					_, err = client.SetupTest(e.newContext(config), &api.SetupTestRequest{
						Suite: suite.Name,
						Test:  test.Name,
					})
					if err != nil {
						return err
					}

					c.Status().Setf("Running '%s'", test.Name)
					response, err := client.RunTest(e.newContext(config), &api.RunTestRequest{
						Suite: suite.Name,
						Test:  test.Name,
					})
					if err != nil {
						return err
					}

					c.Status().Setf("Tearing down '%s'", test.Name)
					_, _ = client.TearDownTest(e.newContext(config), &api.TearDownTestRequest{
						Suite: suite.Name,
						Test:  test.Name,
					})
					if err != nil {
						return err
					}

					if !response.Succeeded {
						return errors.New("test failed")
					}
					return nil
				}
			}

			c.Status().Set("Running tests")
			var testErr error
			if config.Parallel {
				var futures []task.Future
				for j, test := range suite.Tests {
					futures = append(futures, tests[j].Start(newTestRunner(test)))
				}
				err := task.Await(futures...)
				if testErr == nil {
					testErr = err
				}
			} else {
				for j, test := range suite.Tests {
					err := tests[j].Run(newTestRunner(test))
					if testErr == nil {
						testErr = err
					}
				}
			}

			c.Status().Setf("Tearing down '%s'", suite.Name)
			_, err = client.TearDownTestSuite(e.newContext(config), &api.TearDownTestSuiteRequest{
				Suite: suite.Name,
			})
			if testErr == nil {
				testErr = err
			}
			return testErr
		})
		if err != nil {
			testErr = err
		}
	}

	for _, stream := range streams {
		_ = stream.Close()
	}

	err = task.New(e.writer, "Tear down test workers").Run(func(context task.Context) error {
		var futures []task.Future
		for i := 0; i < config.Workers; i++ {
			futures = append(futures, func(worker int) task.Future {
				return context.NewTask("Tear down test worker %d", worker).Start(func(context task.Context) error {
					if err := e.jobs.Stop(jobs[worker], context.Status()); err != nil {
						return err
					}
					return nil
				})
			}(i))
		}
		return task.Await(futures...)
	})
	if testErr == nil {
		testErr = err
	}
	return testErr
}

func newWorkerName(jobID string, worker int) string {
	return fmt.Sprintf("%s-worker-%d", jobID, worker)
}

func (e *testExecutor) getWorkerAddress(config Config, worker int) string {
	return fmt.Sprintf("%s:%d", newWorkerName(e.spec.ID, worker), managementPort)
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

type contextReader struct {
	reader io.Reader
	ctx    context.Context
}

func (r *contextReader) Read(p []byte) (n int, err error) {
	select {
	case <-r.ctx.Done():
		err := r.ctx.Err()
		if err == nil || err == context.Canceled {
			return 0, io.EOF
		}
		return 0, err
	default:
		return r.reader.Read(p)
	}
}

func copyContext(ctx context.Context, writer io.Writer, reader io.Reader) (int64, error) {
	return io.Copy(writer, &contextReader{
		reader: reader,
		ctx:    ctx,
	})
}
