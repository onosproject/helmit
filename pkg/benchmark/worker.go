// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package benchmark

import (
	"context"
	"fmt"
	"github.com/onosproject/helmit/pkg/helm"
	"github.com/onosproject/helmit/pkg/util/logging"
	"google.golang.org/grpc"
	"net"
	"reflect"
	"regexp"
)

// newWorker returns a new benchmark worker
func newWorker(suites map[string]BenchmarkingSuite, config *Config) (*Worker, error) {
	return &Worker{
		suites: suites,
		config: config,
	}, nil
}

// Worker runs a benchmark job
type Worker struct {
	suites map[string]BenchmarkingSuite
	config *Config
}

// Run runs a benchmark
func (w *Worker) Run() error {
	lis, err := net.Listen("tcp", ":5000")
	if err != nil {
		return err
	}
	server := grpc.NewServer()
	RegisterWorkerServiceServer(server, w)
	return server.Serve(lis)
}

func (w *Worker) getSuite(name string) (BenchmarkingSuite, error) {
	if suite, ok := w.suites[name]; ok {
		return suite, nil
	}
	suite, ok := w.suites[name]
	if !ok {
		return nil, fmt.Errorf("unknown benchmark suite %s", name)
	}
	return suite, nil
}

// SetupSuite sets up a benchmark suite
func (w *Worker) SetupSuite(ctx context.Context, request *SuiteRequest) (*SuiteResponse, error) {
	step := logging.NewStep(fmt.Sprintf("%s/%d", request.Suite, getBenchmarkWorker()), "SetupSuite %s", request.Suite)
	step.Start()

	suite, err := w.getSuite(request.Suite)
	if err != nil {
		step.Fail(err)
		return nil, err
	}

	suite.SetHelm(helm.NewClient(helm.Context{
		Namespace:  w.config.Namespace,
		WorkDir:    w.config.Context,
		Values:     w.config.Values,
		ValueFiles: w.config.ValueFiles,
	}))

	ctx, cancel := context.WithTimeout(context.Background(), w.config.Timeout)
	defer cancel()
	for key, value := range request.Args {
		ctx = context.WithValue(ctx, key, value)
	}
	suite.SetContext(ctx)

	if setupSuite, ok := suite.(SetupSuite); ok {
		if err := setupSuite.SetupSuite(); err != nil {
			step.Fail(err)
			return nil, err
		}
	}

	step.Complete()
	return &SuiteResponse{}, nil
}

// TearDownSuite tears down a benchmark suite
func (w *Worker) TearDownSuite(ctx context.Context, request *SuiteRequest) (*SuiteResponse, error) {
	step := logging.NewStep(fmt.Sprintf("%s/%d", request.Suite, getBenchmarkWorker()), "TearDownSuite %s", request.Suite)
	step.Start()

	suite, err := w.getSuite(request.Suite)
	if err != nil {
		step.Fail(err)
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), w.config.Timeout)
	defer cancel()
	for key, value := range request.Args {
		ctx = context.WithValue(ctx, key, value)
	}
	suite.SetContext(ctx)

	if tearDownSuite, ok := suite.(TearDownSuite); ok {
		if err := tearDownSuite.TearDownSuite(); err != nil {
			step.Fail(err)
			return nil, err
		}
	}

	step.Complete()
	return &SuiteResponse{}, nil
}

// SetupWorker sets up a benchmark worker
func (w *Worker) SetupWorker(ctx context.Context, request *SuiteRequest) (*SuiteResponse, error) {
	step := logging.NewStep(fmt.Sprintf("%s/%d", request.Suite, getBenchmarkWorker()), "SetupWorker %s", request.Suite)
	step.Start()

	suite, err := w.getSuite(request.Suite)
	if err != nil {
		step.Fail(err)
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), w.config.Timeout)
	defer cancel()
	for key, value := range request.Args {
		ctx = context.WithValue(ctx, key, value)
	}
	suite.SetContext(ctx)

	if setupWorker, ok := suite.(SetupWorker); ok {
		if err := setupWorker.SetupWorker(); err != nil {
			step.Fail(err)
			return nil, err
		}
	}

	step.Complete()
	return &SuiteResponse{}, nil
}

// TearDownWorker tears down a benchmark worker
func (w *Worker) TearDownWorker(ctx context.Context, request *SuiteRequest) (*SuiteResponse, error) {
	step := logging.NewStep(fmt.Sprintf("%s/%d", request.Suite, getBenchmarkWorker()), "TearDownWorker %s", request.Suite)
	step.Start()

	suite, err := w.getSuite(request.Suite)
	if err != nil {
		step.Fail(err)
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), w.config.Timeout)
	defer cancel()
	for key, value := range request.Args {
		ctx = context.WithValue(ctx, key, value)
	}
	suite.SetContext(ctx)

	if tearDownWorker, ok := suite.(TearDownWorker); ok {
		if err := tearDownWorker.TearDownWorker(); err != nil {
			step.Fail(err)
			return nil, err
		}
	}

	step.Complete()
	return &SuiteResponse{}, nil
}

// SetupBenchmark sets up a benchmark
func (w *Worker) SetupBenchmark(ctx context.Context, request *BenchmarkRequest) (*BenchmarkResponse, error) {
	step := logging.NewStep(fmt.Sprintf("%s/%d", request.Suite, getBenchmarkWorker()), "SetupBenchmark %s", request.Benchmark)
	step.Start()

	suite, err := w.getSuite(request.Suite)
	if err != nil {
		step.Fail(err)
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), w.config.Timeout)
	defer cancel()
	for key, value := range request.Args {
		ctx = context.WithValue(ctx, key, value)
	}
	suite.SetContext(ctx)

	if setupBenchmark, ok := suite.(SetupBenchmark); ok {
		if err := setupBenchmark.SetupBenchmark(request.Suite, request.Benchmark); err != nil {
			step.Fail(err)
			return nil, err
		}
	}

	methods := reflect.TypeOf(suite)
	if method, ok := methods.MethodByName("Setup" + request.Benchmark); ok {
		method.Func.Call([]reflect.Value{reflect.ValueOf(suite)})
	}

	step.Complete()
	return &BenchmarkResponse{}, nil
}

// TearDownBenchmark tears down a benchmark
func (w *Worker) TearDownBenchmark(ctx context.Context, request *BenchmarkRequest) (*BenchmarkResponse, error) {
	step := logging.NewStep(fmt.Sprintf("%s/%d", request.Suite, getBenchmarkWorker()), "TearDownBenchmark %s", request.Benchmark)
	step.Start()

	suite, err := w.getSuite(request.Suite)
	if err != nil {
		step.Fail(err)
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), w.config.Timeout)
	defer cancel()
	for key, value := range request.Args {
		ctx = context.WithValue(ctx, key, value)
	}
	suite.SetContext(ctx)

	if tearDownBenchmark, ok := suite.(TearDownBenchmark); ok {
		if err := tearDownBenchmark.TearDownBenchmark(request.Suite, request.Benchmark); err != nil {
			step.Fail(err)
			return nil, err
		}
	}

	methods := reflect.TypeOf(suite)
	if method, ok := methods.MethodByName("TearDown" + request.Benchmark); ok {
		method.Func.Call([]reflect.Value{reflect.ValueOf(suite)})
	}

	step.Complete()
	return &BenchmarkResponse{}, nil
}

// RunBenchmark runs a benchmark
func (w *Worker) RunBenchmark(ctx context.Context, request *RunRequest) (*RunResponse, error) {
	step := logging.NewStep(fmt.Sprintf("%s/%d", request.Suite, getBenchmarkWorker()), "RunBenchmark %s", request.Benchmark)
	step.Start()

	suite, err := w.getSuite(request.Suite)
	if err != nil {
		step.Fail(err)
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), w.config.Timeout)
	defer cancel()
	for key, value := range request.Args {
		ctx = context.WithValue(ctx, key, value)
	}
	suite.SetContext(ctx)

	benchmark := newBenchmark(request.Benchmark, int(request.Requests), request.Duration, int(request.Parallelism), request.MaxLatency)
	result, err := benchmark.run(suite)
	if err != nil {
		return nil, err
	}
	step.Complete()
	return result, nil
}

// benchmarkFilter filters benchmark method names
func benchmarkFilter(name string) (bool, error) {
	if ok, _ := regexp.MatchString("^B", name); !ok {
		return false, nil
	}
	return true, nil
}
