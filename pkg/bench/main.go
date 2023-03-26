// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package bench

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/onosproject/helmit/internal/job"
	"github.com/onosproject/helmit/pkg/helm"
	"k8s.io/client-go/rest"
	"math"
	"os"
	"reflect"
	"sort"
	"sync/atomic"
	"time"
)

const shutdownFile = "/tmp/shutdown"

// The executor is the entrypoint for benchmark images. It takes the input and environment and runs
// the image in the appropriate context according to the arguments.

// Main runs a benchmark
func Main(suites map[string]BenchmarkingSuite) {
	if err := run(suites); err != nil {
		println("Benchmark failed " + err.Error())
		os.Exit(1)
	}
	os.Exit(0)
}

// run runs a benchmark
func run(suites map[string]BenchmarkingSuite) error {
	var config Config
	if err := job.Bootstrap(&config); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	ctx := context.Background()
	for key, value := range config.Args {
		ctx = context.WithValue(ctx, key, value)
	}

	suite, ok := suites[config.Suite]
	if !ok {
		return fmt.Errorf("unknown suite %s", config.Suite)
	}

	suite.SetNamespace(config.Namespace)
	raftConfig, err := rest.InClusterConfig()
	if err != nil {
		return err
	}
	suite.SetConfig(raftConfig)

	suite.SetHelm(helm.NewClient(helm.Context{
		Namespace:  config.Namespace,
		WorkDir:    config.Context,
		Values:     config.Values,
		ValueFiles: config.ValueFiles,
	}))

	methodFinder := reflect.TypeOf(suite)
	method, ok := methodFinder.MethodByName(config.Benchmark)
	if !ok {
		return fmt.Errorf("unknown benchmark %s", config.Benchmark)
	}

	switch config.Type {
	case SetupType:
		if setupSuite, ok := suite.(SetupSuite); ok {
			ctx, cancel := context.WithTimeout(ctx, config.Timeout)
			defer cancel()
			if err := setupSuite.SetupSuite(ctx); err != nil {
				return err
			}
		}
		if setupBench, ok := suite.(SetupBenchmark); ok {
			ctx, cancel := context.WithTimeout(ctx, config.Timeout)
			defer cancel()
			if err := setupBench.SetupBenchmark(ctx); err != nil {
				return err
			}
		}
		if setupMethod, ok := methodFinder.MethodByName("Setup" + method.Name); ok {
			ctx, cancel := context.WithTimeout(ctx, config.Timeout)
			defer cancel()
			values := setupMethod.Func.Call([]reflect.Value{reflect.ValueOf(suite), reflect.ValueOf(ctx)})
			if len(values) > 0 {
				value := values[0]
				if !value.IsNil() {
					return value.Interface().(error)
				}
			}
		}
	case WorkerType:
		if setupWorker, ok := suite.(SetupWorker); ok {
			ctx, cancel := context.WithTimeout(ctx, config.Timeout)
			defer cancel()
			if err := setupWorker.SetupWorker(ctx); err != nil {
				return err
			}
		}

		f := func() error {
			ctx, cancel := context.WithTimeout(ctx, config.Timeout)
			defer cancel()
			values := method.Func.Call([]reflect.Value{reflect.ValueOf(suite), reflect.ValueOf(ctx)})
			if len(values) == 0 || values[0].Interface() == nil {
				return nil
			}
			return values[0].Interface().(error)
		}

		shutdownCh := make(chan struct{})
		go func() {
			awaitShutdown()
			close(shutdownCh)
		}()

		stopped := &atomic.Bool{}
		results := make(chan time.Duration, 1000)
		for i := 0; i < config.Parallelism; i++ {
			go func() {
				for !stopped.Load() {
					start := time.Now()
					if err := f(); err == nil {
						results <- time.Since(start)
					}
				}
			}()
		}

		ticker := time.NewTicker(config.ReportInterval)
		start := time.Now()
		var calls []time.Duration
		for {
			select {
			case <-ticker.C:
				sort.Slice(calls, func(i, j int) bool {
					return calls[i] < calls[j]
				})

				// Calculate the total latency from latency results
				var totalCallRTT time.Duration
				for _, rtt := range calls {
					totalCallRTT += rtt
				}

				// Compute the report statistics
				report := Report{
					Iterations:  len(calls),
					Duration:    time.Since(start),
					MeanLatency: time.Duration(int64(totalCallRTT) / int64(len(calls))),
					P50Latency:  calls[int(math.Max(float64(len(calls)/2)-1, 0))],
					P75Latency:  calls[int(math.Max(float64(len(calls)-(len(calls)/4)-1), 0))],
					P95Latency:  calls[int(math.Max(float64(len(calls)-(len(calls)/20)-1), 0))],
					P99Latency:  calls[int(math.Max(float64(len(calls)-(len(calls)/100)-1), 0))],
				}

				bytes, err := json.Marshal(&report)
				if err != nil {
					return err
				}
				fmt.Println(string(bytes))

				start = time.Now()
				calls = []time.Duration{}
			case result := <-results:
				calls = append(calls, result)
			case <-shutdownCh:
				stopped.Store(true)
				if tearDownWorker, ok := suite.(TearDownWorker); ok {
					ctx, cancel := context.WithTimeout(ctx, config.Timeout)
					defer cancel()
					if err := tearDownWorker.TearDownWorker(ctx); err != nil {
						return err
					}
				}
			}
		}
	case TearDownType:
		if tearDownMethod, ok := methodFinder.MethodByName("TearDown" + method.Name); ok {
			ctx, cancel := context.WithTimeout(ctx, config.Timeout)
			defer cancel()
			values := tearDownMethod.Func.Call([]reflect.Value{reflect.ValueOf(suite), reflect.ValueOf(ctx)})
			if len(values) > 0 {
				value := values[0]
				if !value.IsNil() {
					return value.Interface().(error)
				}
			}
		}
		if tearDownBench, ok := suite.(TearDownBenchmark); ok {
			ctx, cancel := context.WithTimeout(ctx, config.Timeout)
			defer cancel()
			if err := tearDownBench.TearDownBenchmark(ctx); err != nil {
				return err
			}
		}
		if tearDownSuite, ok := suite.(TearDownSuite); ok {
			ctx, cancel := context.WithTimeout(ctx, config.Timeout)
			defer cancel()
			if err := tearDownSuite.TearDownSuite(ctx); err != nil {
				return err
			}
		}
	}
	return nil
}

func awaitShutdown() {
	for {
		if isShutdown() {
			return
		}
		time.Sleep(time.Second)
	}
}

func isShutdown() bool {
	info, err := os.Stat(shutdownFile)
	return err == nil && !info.IsDir()
}

type Report struct {
	Iterations  int           `json:"iterations"`
	Duration    time.Duration `json:"duration"`
	MeanLatency time.Duration `json:"meanLatency"`
	P50Latency  time.Duration `json:"p50Latency"`
	P75Latency  time.Duration `json:"p75Latency"`
	P95Latency  time.Duration `json:"p95Latency"`
	P99Latency  time.Duration `json:"p99Latency"`
}
