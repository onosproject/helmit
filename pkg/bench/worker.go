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
	"regexp"
	"sort"
	"sync"
	"time"
)

// newWorker returns a new benchmark worker
func newWorker(spec job.Spec, suites map[string]BenchmarkingSuite) (*Worker, error) {
	return &Worker{
		spec:   spec,
		suites: suites,
	}, nil
}

// Worker runs a benchmark job
type Worker struct {
	spec   job.Spec
	suites map[string]BenchmarkingSuite
}

// Run runs a benchmark
func (w *Worker) Run() error {
	lis, err := net.Listen("tcp", ":5000")
	if err != nil {
		return err
	}
	server := grpc.NewServer()
	api.RegisterBenchmarkerServer(server, w)
	return server.Serve(lis)
}

// SetupBenchmarkSuite sets up a benchmark suite
func (w *Worker) SetupBenchmarkSuite(ctx context.Context, request *api.SetupBenchmarkSuiteRequest) (*api.SetupBenchmarkSuiteResponse, error) {
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
func (w *Worker) TearDownBenchmarkSuite(ctx context.Context, request *api.TearDownBenchmarkSuiteRequest) (*api.TearDownBenchmarkSuiteResponse, error) {
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
func (w *Worker) SetupBenchmarkWorker(ctx context.Context, request *api.SetupBenchmarkWorkerRequest) (*api.SetupBenchmarkWorkerResponse, error) {
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
func (w *Worker) TearDownBenchmarkWorker(ctx context.Context, request *api.TearDownBenchmarkWorkerRequest) (*api.TearDownBenchmarkWorkerResponse, error) {
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
func (w *Worker) SetupBenchmark(ctx context.Context, request *api.SetupBenchmarkRequest) (*api.SetupBenchmarkResponse, error) {
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
func (w *Worker) TearDownBenchmark(ctx context.Context, request *api.TearDownBenchmarkRequest) (*api.TearDownBenchmarkResponse, error) {
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

// RunBenchmark runs a benchmark
func (w *Worker) RunBenchmark(ctx context.Context, request *api.RunBenchmarkRequest) (*api.RunBenchmarkResponse, error) {
	suite, ok := w.suites[request.Suite]
	if !ok {
		return nil, errors.NewNotFound("unknown suite %s", request.Suite)
	}

	suite.SetB(&B{
		Suite:  request.Suite,
		Name:   request.Benchmark,
		Worker: getBenchmarkWorker(),
		out:    os.Stdout,
	})

	var statistics api.BenchmarkStatistics

	var f func(context.Context) error
	methods := reflect.TypeOf(suite)
	if method, ok := methods.MethodByName(request.Benchmark); ok {
		f = func(context.Context) error {
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

	ctx, cancel := context.WithTimeout(ctx, w.spec.Timeout)
	defer cancel()

	// Run the benchmark
	requests, runTime, results := w.runBenchmark(ctx, request, f)

	// Calculate the total latency from latency results
	var totalLatency time.Duration
	for _, result := range results {
		totalLatency += result
	}

	// Compute statistics
	statistics.Iterations = uint64(requests)
	statistics.Duration = types.DurationProto(runTime)
	statistics.MeanLatency = types.DurationProto(time.Duration(int64(totalLatency) / int64(len(results))))
	statistics.P50Latency = types.DurationProto(results[int(math.Max(float64(len(results)/2)-1, 0))])
	statistics.P75Latency = types.DurationProto(results[int(math.Max(float64(len(results)-(len(results)/4)-1), 0))])
	statistics.P95Latency = types.DurationProto(results[int(math.Max(float64(len(results)-(len(results)/20)-1), 0))])
	statistics.P99Latency = types.DurationProto(results[int(math.Max(float64(len(results)-(len(results)/100)-1), 0))])

	return &api.RunBenchmarkResponse{
		Statistics: &statistics,
	}, nil
}

// warmBenchmark runs the benchmark
func (w *Worker) runBenchmark(ctx context.Context, request *api.RunBenchmarkRequest, benchmark func(context.Context) error) (int, time.Duration, []time.Duration) {
	// Create an iteration channel and wait group and create a goroutine for each client
	wg := &sync.WaitGroup{}
	parallelism := int(request.Parallelism)
	requestCh := make(chan struct{}, parallelism)
	resultCh := make(chan time.Duration, aggBatchSize)
	for i := 0; i < parallelism; i++ {
		wg.Add(1)
		go func() {
			for range requestCh {
				start := time.Now()
				ctx, cancel := context.WithCancel(ctx)
				_ = benchmark(ctx)
				cancel()
				end := time.Now()
				resultCh <- end.Sub(start)
			}
			wg.Done()
		}()
	}

	// Start an aggregator goroutine
	results := make([]time.Duration, 0, aggBatchSize*aggBatchSize)
	aggWg := &sync.WaitGroup{}
	aggWg.Add(1)
	go func() {
		var total time.Duration
		var count = 0
		// Iterate through results and aggregate durations
		for duration := range resultCh {
			total += duration
			count++
			// Average out the durations in batches
			if count == aggBatchSize {
				results = append(results, total/time.Duration(count))

				// If the total number of batches reaches the batch size ^ 2, aggregate the aggregated results
				if len(results) == aggBatchSize*aggBatchSize {
					newResults := make([]time.Duration, 0, aggBatchSize*aggBatchSize)
					for _, result := range results {
						total += result
						count++
						if count == aggBatchSize {
							newResults = append(newResults, total/time.Duration(count))
							total = 0
							count = 0
						}
					}
					results = newResults
				}
				total = 0
				count = 0
			}
		}
		if count > 0 {
			results = append(results, total/time.Duration(count))
		}
		aggWg.Done()
	}()

	// Record the start time and write arguments to the channel
	start := time.Now()

	var iterations uint64
	if request.Iterations != 0 {
		for request.Iterations == 0 || iterations < request.Iterations {
			requestCh <- struct{}{}
			iterations++
		}
	} else {
		var duration *time.Duration
		if request.Duration != nil {
			if d, err := types.DurationFromProto(request.Duration); err == nil {
				duration = &d
			} else {
				panic(err)
			}
		}

		for request.Duration == nil || time.Since(start) < *duration {
			requestCh <- struct{}{}
			iterations++
		}
	}
	close(requestCh)

	// Wait for the tests to finish and close the result channel
	wg.Wait()

	// Record the end time
	duration := time.Since(start)

	// Close the output channel
	close(resultCh)

	// Wait for the results to be aggregated
	aggWg.Wait()

	// Sort the aggregated results
	sort.Slice(results, func(i, j int) bool {
		return results[i] < results[j]
	})
	return int(iterations), duration, results
}

// benchmarkFilter filters benchmark method names
func benchmarkFilter(name string) (bool, error) {
	if ok, _ := regexp.MatchString("^B", name); !ok {
		return false, nil
	}
	return true, nil
}
