// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package benchmark

import (
	"context"
	"fmt"
	"github.com/onosproject/helmit/pkg/helm"
	"math"
	"reflect"
	"sort"
	"sync"
	"time"
)

const warmUpDuration = 30 * time.Second
const aggBatchSize = 100

// BenchmarkingSuite is a suite of benchmarks
type BenchmarkingSuite interface {
	SetB(b *B)
	B() *B
	SetHelm(helm *helm.Helm)
	Helm() *helm.Helm
	SetContext(ctx context.Context)
	Context() context.Context
}

// Suite is the base for a benchmark suite
type Suite struct {
	b    *B
	helm *helm.Helm
	ctx  context.Context
}

func (suite *Suite) Namespace() string {
	return suite.helm.Namespace()
}

func (suite *Suite) SetB(b *B) {
	suite.b = b
}

func (suite *Suite) B() *B {
	return suite.b
}

func (suite *Suite) SetHelm(helm *helm.Helm) {
	suite.helm = helm
}

func (suite *Suite) Helm() *helm.Helm {
	return suite.helm
}

func (suite *Suite) SetContext(ctx context.Context) {
	suite.ctx = ctx
}

func (suite *Suite) Context() context.Context {
	return suite.ctx
}

// SetupSuite is an interface for setting up a suite of benchmarks
type SetupSuite interface {
	SetupSuite() error
}

// TearDownSuite is an interface for tearing down a suite of benchmarks
type TearDownSuite interface {
	TearDownSuite() error
}

// SetupWorker is an interface for setting up individual benchmarks
type SetupWorker interface {
	SetupWorker() error
}

// TearDownWorker is an interface for tearing down individual benchmarks
type TearDownWorker interface {
	TearDownWorker() error
}

// SetupBenchmark is an interface for executing code before every benchmark
type SetupBenchmark interface {
	SetupBenchmark(suite string, name string) error
}

// TearDownBenchmark is an interface for executing code after every benchmark
type TearDownBenchmark interface {
	TearDownBenchmark(suite string, name string) error
}

// newBenchmark creates a new benchmark
func newBenchmark(name string, requests int, duration *time.Duration, parallelism int, maxLatency *time.Duration) *B {
	return &B{
		Name:        name,
		requests:    requests,
		duration:    duration,
		maxLatency:  maxLatency,
		parallelism: parallelism,
	}
}

// B is a benchmark runner
type B struct {
	Name        string
	requests    int
	duration    *time.Duration
	parallelism int
	maxLatency  *time.Duration
}

// Run runs the benchmark with the given parameters
func (b *B) run(suite BenchmarkingSuite) (*RunResponse, error) {
	var f func() error
	methods := reflect.TypeOf(suite)
	if method, ok := methods.MethodByName(b.Name); ok {
		f = func() error {
			values := method.Func.Call([]reflect.Value{reflect.ValueOf(suite)})
			if len(values) == 0 {
				return nil
			} else if values[0].Interface() == nil {
				return nil
			}
			return values[0].Interface().(error)
		}
	} else {
		return nil, fmt.Errorf("unknown benchmark method %s", b.Name)
	}

	// Warm the benchmark
	b.warmRequests(f)

	// Run the benchmark
	requests, runTime, results := b.runRequests(f)

	// Calculate the total latency from latency results
	var totalLatency time.Duration
	for _, result := range results {
		totalLatency += result
	}

	// Calculate latency percentiles
	meanLatency := time.Duration(int64(totalLatency) / int64(len(results)))
	latency50 := results[int(math.Max(float64(len(results)/2)-1, 0))]
	latency75 := results[int(math.Max(float64(len(results)-(len(results)/4)-1), 0))]
	latency95 := results[int(math.Max(float64(len(results)-(len(results)/20)-1), 0))]
	latency99 := results[int(math.Max(float64(len(results)-(len(results)/100)-1), 0))]

	return &RunResponse{
		Requests:  uint32(requests),
		Duration:  runTime,
		Latency:   meanLatency,
		Latency50: latency50,
		Latency75: latency75,
		Latency95: latency95,
		Latency99: latency99,
	}, nil
}

// warm warms up the benchmark
func (b *B) warmRequests(f func() error) {
	// Create an iteration channel and wait group and create a goroutine for each client
	wg := &sync.WaitGroup{}
	requestCh := make(chan struct{}, b.parallelism)
	for i := 0; i < b.parallelism; i++ {
		wg.Add(1)
		go func() {
			for range requestCh {
				_ = f()
			}
			wg.Done()
		}()
	}

	// Run for the warm up duration to prepare the benchmark
	start := time.Now()
	for time.Since(start) < warmUpDuration {
		requestCh <- struct{}{}
	}
	close(requestCh)

	// Wait for the tests to finish and close the result channel
	wg.Wait()
}

// run runs the benchmark
func (b *B) runRequests(f func() error) (int, time.Duration, []time.Duration) {
	// Create an iteration channel and wait group and create a goroutine for each client
	wg := &sync.WaitGroup{}
	requestCh := make(chan struct{}, b.parallelism)
	resultCh := make(chan time.Duration, aggBatchSize)
	for i := 0; i < b.parallelism; i++ {
		wg.Add(1)
		go func() {
			for range requestCh {
				start := time.Now()
				_ = f()
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

	// Iterate through the request count or until the time duration has been met
	requests := 0
	for (b.requests == 0 || requests < b.requests) && (b.duration == nil || time.Since(start) < *b.duration) {
		requestCh <- struct{}{}
		requests++
	}
	close(requestCh)

	// Wait for the tests to finish and close the result channel
	wg.Wait()

	// Record the end time
	end := time.Now()
	duration := end.Sub(start)

	// Close the output channel
	close(resultCh)

	// Wait for the results to be aggregated
	aggWg.Wait()

	// Sort the aggregated results
	sort.Slice(results, func(i, j int) bool {
		return results[i] < results[j]
	})
	return requests, duration, results
}

// getBenchmarks returns a list of benchmarks in the given suite
func getBenchmarks(suite BenchmarkingSuite) []string {
	methodFinder := reflect.TypeOf(suite)
	benchmarks := []string{}
	for index := 0; index < methodFinder.NumMethod(); index++ {
		method := methodFinder.Method(index)
		ok, err := benchmarkFilter(method.Name)
		if ok {
			benchmarks = append(benchmarks, method.Name)
		} else if err != nil {
			panic(err)
		}
	}
	return benchmarks
}
