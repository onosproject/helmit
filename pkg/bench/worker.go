// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package bench

import (
	"context"
	"fmt"
	"github.com/gogo/protobuf/types"
	api "github.com/onosproject/helmit/api/v1"
	"github.com/onosproject/helmit/internal/job"
	"github.com/onosproject/helmit/internal/k8s"
	"github.com/onosproject/helmit/pkg/helm"
	"github.com/onosproject/onos-lib-go/pkg/errors"
	"google.golang.org/grpc"
	"math"
	"net"
	"os"
	"reflect"
	"sync"
	"sync/atomic"
	"time"
)

// newWorker returns a new benchmark worker
func newWorker(spec job.Spec, suites map[string]BenchmarkingSuite) (*benchWorker, error) {
	return &benchWorker{
		spec:   spec,
		suites: suites,
	}, nil
}

// benchWorker runs a benchmark job
type benchWorker struct {
	spec    job.Spec
	suites  map[string]BenchmarkingSuite
	start   time.Time
	calls   []time.Duration
	stopped atomic.Bool
	mu      sync.Mutex
}

// run runs a benchmark
func (w *benchWorker) run() error {
	lis, err := net.Listen("tcp", ":5000")
	if err != nil {
		return err
	}
	server := grpc.NewServer()
	api.RegisterBenchmarkerServer(server, w)
	return server.Serve(lis)
}

// SetupBenchmarkSuite sets up a benchmark suite
func (w *benchWorker) SetupBenchmarkSuite(ctx context.Context, request *api.SetupBenchmarkSuiteRequest) (*api.SetupBenchmarkSuiteResponse, error) {
	suite, ok := w.suites[request.Suite]
	if !ok {
		return nil, errors.NewNotFound("unknown suite %s", request.Suite)
	}

	config, err := k8s.GetConfig()
	if err != nil {
		return nil, err
	}
	suite.SetConfig(config)

	suite.SetHelm(helm.NewClient(helm.Context{
		Namespace:  w.spec.Namespace,
		WorkDir:    w.spec.Context,
		Values:     w.spec.Values,
		ValueFiles: w.spec.ValueFiles,
	}))

	if setupSuite, ok := suite.(SetupSuite); ok {
		ctx, cancel := context.WithTimeout(ctx, w.spec.Timeout)
		defer cancel()
		if err := setupSuite.SetupSuite(ctx); err != nil {
			return nil, err
		}
	}
	return &api.SetupBenchmarkSuiteResponse{}, nil
}

// TearDownBenchmarkSuite tears down a benchmark suite
func (w *benchWorker) TearDownBenchmarkSuite(ctx context.Context, request *api.TearDownBenchmarkSuiteRequest) (*api.TearDownBenchmarkSuiteResponse, error) {
	suite, ok := w.suites[request.Suite]
	if !ok {
		return nil, errors.NewNotFound("unknown suite %s", request.Suite)
	}

	if tearDownSuite, ok := suite.(TearDownSuite); ok {
		ctx, cancel := context.WithTimeout(ctx, w.spec.Timeout)
		defer cancel()
		if err := tearDownSuite.TearDownSuite(ctx); err != nil {
			return nil, err
		}
	}
	return &api.TearDownBenchmarkSuiteResponse{}, nil
}

// SetupBenchmarkWorker sets up a benchmark worker
func (w *benchWorker) SetupBenchmarkWorker(ctx context.Context, request *api.SetupBenchmarkWorkerRequest) (*api.SetupBenchmarkWorkerResponse, error) {
	suite, ok := w.suites[request.Suite]
	if !ok {
		return nil, errors.NewNotFound("unknown suite %s", request.Suite)
	}

	if setupWorker, ok := suite.(SetupWorker); ok {
		ctx, cancel := context.WithTimeout(ctx, w.spec.Timeout)
		defer cancel()
		if err := setupWorker.SetupWorker(ctx); err != nil {
			return nil, err
		}
	}
	return &api.SetupBenchmarkWorkerResponse{}, nil
}

// TearDownBenchmarkWorker tears down a benchmark worker
func (w *benchWorker) TearDownBenchmarkWorker(ctx context.Context, request *api.TearDownBenchmarkWorkerRequest) (*api.TearDownBenchmarkWorkerResponse, error) {
	suite, ok := w.suites[request.Suite]
	if !ok {
		return nil, errors.NewNotFound("unknown suite %s", request.Suite)
	}

	if tearDownWorker, ok := suite.(TearDownWorker); ok {
		ctx, cancel := context.WithTimeout(ctx, w.spec.Timeout)
		defer cancel()
		if err := tearDownWorker.TearDownWorker(ctx); err != nil {
			return nil, err
		}
	}
	return &api.TearDownBenchmarkWorkerResponse{}, nil
}

// SetupBenchmark sets up a benchmark
func (w *benchWorker) SetupBenchmark(ctx context.Context, request *api.SetupBenchmarkRequest) (*api.SetupBenchmarkResponse, error) {
	suite, ok := w.suites[request.Suite]
	if !ok {
		return nil, errors.NewNotFound("unknown suite %s", request.Suite)
	}

	if setupBenchmark, ok := suite.(SetupBenchmark); ok {
		ctx, cancel := context.WithTimeout(ctx, w.spec.Timeout)
		defer cancel()
		if err := setupBenchmark.SetupBenchmark(ctx, request.Suite, request.Benchmark); err != nil {
			return nil, err
		}
	}

	methods := reflect.TypeOf(suite)
	if method, ok := methods.MethodByName("Setup" + request.Benchmark); ok {
		method.Func.Call([]reflect.Value{reflect.ValueOf(suite)})
	}
	return &api.SetupBenchmarkResponse{}, nil
}

// TearDownBenchmark tears down a benchmark
func (w *benchWorker) TearDownBenchmark(ctx context.Context, request *api.TearDownBenchmarkRequest) (*api.TearDownBenchmarkResponse, error) {
	suite, ok := w.suites[request.Suite]
	if !ok {
		return nil, errors.NewNotFound("unknown suite %s", request.Suite)
	}

	if tearDownBenchmark, ok := suite.(TearDownBenchmark); ok {
		ctx, cancel := context.WithTimeout(ctx, w.spec.Timeout)
		defer cancel()
		if err := tearDownBenchmark.TearDownBenchmark(ctx, request.Suite, request.Benchmark); err != nil {
			return nil, err
		}
	}

	methods := reflect.TypeOf(suite)
	if method, ok := methods.MethodByName("TearDown" + request.Benchmark); ok {
		method.Func.Call([]reflect.Value{reflect.ValueOf(suite)})
	}
	return &api.TearDownBenchmarkResponse{}, nil
}

// StartBenchmark starts a benchmark
func (w *benchWorker) StartBenchmark(ctx context.Context, request *api.StartBenchmarkRequest) (*api.StartBenchmarkResponse, error) {
	suite, ok := w.suites[request.Suite]
	if !ok {
		return nil, errors.NewNotFound("unknown suite %s", request.Suite)
	}

	suite.SetB(&B{
		Suite: request.Suite,
		Name:  request.Benchmark,
		out:   os.Stdout,
	})

	timeout, err := types.DurationFromProto(request.Timeout)
	if err != nil {
		return nil, err
	}

	var f func() error
	methods := reflect.TypeOf(suite)
	if method, ok := methods.MethodByName(request.Benchmark); ok {
		f = func() error {
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()
			values := method.Func.Call([]reflect.Value{reflect.ValueOf(suite), reflect.ValueOf(ctx)})
			if len(values) == 0 {
				return nil
			} else if values[0].Interface() == nil {
				return nil
			}
			return values[0].Interface().(error)
		}
	} else {
		return nil, fmt.Errorf("unknown benchmark method %s", request.Benchmark)
	}

	w.mu.Lock()
	w.start = time.Now()
	w.calls = []time.Duration{}
	w.mu.Unlock()

	parallelism := int(request.Parallelism)
	if parallelism == 0 {
		parallelism = 1
	}
	for i := 0; i < parallelism; i++ {
		go func() {
			for !w.stopped.Load() {
				start := time.Now()
				if err := f(); err == nil {
					duration := time.Since(start)
					go func() {
						w.mu.Lock()
						defer w.mu.Unlock()
						w.calls = append(w.calls, duration)
					}()
				}
			}
		}()
	}
	return &api.StartBenchmarkResponse{}, nil
}

func (w *benchWorker) ReportBenchmark(ctx context.Context, request *api.ReportBenchmarkRequest) (*api.ReportBenchmarkResponse, error) {
	w.mu.Lock()
	duration := time.Since(w.start)
	calls := w.calls
	w.start = time.Now()
	w.calls = []time.Duration{}
	w.mu.Unlock()

	// Calculate the total latency from latency results
	var totalCallRTT time.Duration
	for _, rtt := range calls {
		totalCallRTT += rtt
	}

	// Compute the report statistics
	report := &api.BenchmarkReport{
		Iterations:  uint64(len(calls)),
		Duration:    types.DurationProto(duration),
		MeanLatency: types.DurationProto(time.Duration(int64(totalCallRTT) / int64(len(calls)))),
		P50Latency:  types.DurationProto(calls[int(math.Max(float64(len(calls)/2)-1, 0))]),
		P75Latency:  types.DurationProto(calls[int(math.Max(float64(len(calls)-(len(calls)/4)-1), 0))]),
		P95Latency:  types.DurationProto(calls[int(math.Max(float64(len(calls)-(len(calls)/20)-1), 0))]),
		P99Latency:  types.DurationProto(calls[int(math.Max(float64(len(calls)-(len(calls)/100)-1), 0))]),
	}

	return &api.ReportBenchmarkResponse{
		Report: report,
	}, nil
}

// StopBenchmark stops a benchmark
func (w *benchWorker) StopBenchmark(ctx context.Context, request *api.StopBenchmarkRequest) (*api.StopBenchmarkResponse, error) {
	w.stopped.Store(true)
	return &api.StopBenchmarkResponse{}, nil
}
