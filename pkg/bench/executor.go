// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package bench

import (
	"context"
	"fmt"
	"github.com/gogo/protobuf/types"
	grpc_retry "github.com/grpc-ecosystem/go-grpc-middleware/retry"
	api "github.com/onosproject/helmit/api/v1"
	"github.com/onosproject/helmit/internal/async"
	"github.com/onosproject/helmit/internal/job"
	"github.com/onosproject/helmit/pkg/util/logging"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"math"
	"os"
	"reflect"
	"sync"
	"text/tabwriter"
	"time"
)

// newExecutor returns a new benchmark job
func newExecutor(spec job.Spec, suites map[string]BenchmarkingSuite, tasks *console.ContextManager) (*benchExecutor, error) {
	return &benchExecutor{
		spec:    spec,
		suites:  suites,
		manager: job.NewManager[WorkerConfig](tasks),
		tasks:   tasks,
	}, nil
}

// benchExecutor coordinates workers for suites of benchmarks
type benchExecutor struct {
	spec    job.Spec
	suites  map[string]BenchmarkingSuite
	manager *job.Manager[WorkerConfig]
	tasks   *console.ContextManager
}

// Run runs the tests
func (e *benchExecutor) run(config Config) error {
	if config.Suite == "" {
		for name := range e.suites {
			suite := config
			suite.Suite = name
			executor, err := newSuiteExecutor(e, suite)
			if err != nil {
				return err
			}
			if err := executor.run(); err != nil {
				return err
			}
		}
	} else {
		executor, err := newSuiteExecutor(e, config)
		if err != nil {
			return err
		}
		if err := executor.run(); err != nil {
			return err
		}
	}
	return nil
}

func newSuiteExecutor(executor *benchExecutor, config Config) (*suiteExecutor, error) {
	suite, ok := executor.suites[config.Suite]
	if !ok {
		return nil, fmt.Errorf("unknown suite %s", config.Suite)
	}

	workers := make(map[int]api.BenchmarkerClient)
	for i := 0; i < config.Workers; i++ {
		worker, err := grpc.Dial(
			fmt.Sprintf("%s:5000", newWorkerName(executor.spec.ID, config.Suite, i)),
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithUnaryInterceptor(
				grpc_retry.UnaryClientInterceptor(
					grpc_retry.WithCodes(codes.Unavailable),
					grpc_retry.WithBackoff(grpc_retry.BackoffExponential(1*time.Second)),
					grpc_retry.WithMax(10))))
		if err != nil {
			return nil, err
		}
		workers[i] = api.NewBenchmarkerClient(worker)
	}
	return &suiteExecutor{
		benchExecutor: executor,
		suite:         suite,
		config:        config,
		workers:       workers,
	}, nil
}

type suiteExecutor struct {
	*benchExecutor
	suite   BenchmarkingSuite
	config  Config
	workers map[int]api.BenchmarkerClient
}

// Run runs the tests
func (e *suiteExecutor) run(context *console.Context) error {
	for worker := 0; worker < e.config.Workers; worker++ {
		context.Fork(fmt.Sprintf("Creating worker %d", worker), func(context *console.Context) error {
			return e.createWorker(worker, context)
		})
	}
	if err := context.Wait(); err != nil {
		return err
	}

	if err := e.runBenchmarks(); err != nil {
		return err
	}
	return nil
}

func newWorkerName(jobID string, suite string, worker int) string {
	return fmt.Sprintf("%s-%s-worker-%d", jobID, suite, worker)
}

func (e *suiteExecutor) getWorkerAddress(config Config, worker int) string {
	return fmt.Sprintf("%s:5000", newWorkerName(e.spec.ID, config.Suite, worker))
}

// createWorkers creates the benchmark workers
func (e *suiteExecutor) createWorkers(status *console.TaskReporter) error {
	return async.IterAsync(e.config.Workers, func(i int) error {
		status := status.NewSubTask("Creating worker %d", i)
		if err := e.createWorker(i, status); err != nil {
			status.Error(err)
			return err
		}
		status.Done()
		return nil
	})
}

// createWorker creates the given worker
func (e *suiteExecutor) createWorker(worker int, context *console.Context) error {
	jobID := newWorkerName(e.spec.ID, e.config.Suite, worker)
	env := e.config.WorkerConfig.Env
	if env == nil {
		env = make(map[string]string)
	}
	env[benchmarkTypeEnv] = string(benchTypeWorker)
	env[benchmarkWorkerEnv] = fmt.Sprintf("%d", worker)
	return e.manager.Start(job.Job[WorkerConfig]{
		Spec: job.Spec{
			ID:              jobID,
			Namespace:       e.spec.Namespace,
			ServiceAccount:  e.spec.ServiceAccount,
			Labels:          e.spec.Labels,
			Annotations:     e.spec.Annotations,
			Image:           e.config.WorkerConfig.Image,
			ImagePullPolicy: e.config.WorkerConfig.ImagePullPolicy,
			Executable:      e.spec.Executable,
			Context:         e.spec.Context,
			Values:          e.spec.Values,
			ValueFiles:      e.spec.ValueFiles,
			Env:             env,
			Timeout:         e.spec.Timeout,
			NoTeardown:      e.spec.NoTeardown,
			Secrets:         e.spec.Secrets,
			ManagementPort:  5000,
		},
	}, context)
}

func (e *suiteExecutor) newContext() context.Context {
	ctx := context.Background()
	for key, value := range e.config.Args {
		ctx = context.WithValue(ctx, key, value)
	}
	return ctx
}

// setupSuite sets up the benchmark suite
func (e *suiteExecutor) setupSuite() error {
	var retErr error
	for _, worker := range e.workers {
		_, err := worker.SetupBenchmarkSuite(e.newContext(), &api.SetupBenchmarkSuiteRequest{
			Suite: e.config.Suite,
		})
		if err != nil {
			retErr = err
		} else {
			return nil
		}
	}
	return retErr
}

// setupWorkers sets up the benchmark workers
func (e *suiteExecutor) setupWorkers() error {
	wg := &sync.WaitGroup{}
	errCh := make(chan error)
	for _, worker := range e.workers {
		wg.Add(1)
		go func(worker api.BenchmarkerClient) {
			_, err := worker.SetupBenchmarkWorker(e.newContext(), &api.SetupBenchmarkWorkerRequest{
				Suite: e.config.Suite,
			})
			if err != nil {
				errCh <- err
			}
			wg.Done()
		}(worker)
	}
	wg.Wait()
	close(errCh)

	for err := range errCh {
		return err
	}
	return nil
}

// setupBenchmark sets up the given benchmark
func (e *suiteExecutor) setupBenchmark() error {
	wg := &sync.WaitGroup{}
	errCh := make(chan error)
	for _, worker := range e.workers {
		wg.Add(1)
		go func(worker api.BenchmarkerClient) {
			_, err := worker.SetupBenchmark(e.newContext(), &api.SetupBenchmarkRequest{
				Suite:     e.config.Suite,
				Benchmark: e.config.Benchmark,
			})
			if err != nil {
				errCh <- err
			}
			wg.Done()
		}(worker)
	}
	wg.Wait()
	close(errCh)

	for err := range errCh {
		return err
	}
	return nil
}

// runBenchmarks runs the given benchmarks
func (e *suiteExecutor) runBenchmarks() error {
	// Setup the benchmark suite on one of the workers
	if err := e.setupSuite(); err != nil {
		return err
	}

	// Setup the workers
	if err := e.setupWorkers(); err != nil {
		return err
	}

	// Run the benchmarks
	results := make([]result, 0)
	if e.config.Benchmark != "" {
		step := logging.NewStep(e.spec.ID, "Run benchmark %s", e.config.Benchmark)
		step.Start()
		result, err := e.runBenchmark(e.config)
		if err != nil {
			step.Fail(err)
			return err
		}
		step.Complete()
		results = append(results, result)
	} else {
		suiteStep := logging.NewStep(e.spec.ID, "Run benchmark suite %s", e.config.Suite)
		suiteStep.Start()
		benchmarks := listBenchmarks(e.suite)
		for _, benchmark := range benchmarks {
			benchmarkSuite := logging.NewStep(e.spec.ID, "Run benchmark %s", benchmark)
			benchmarkSuite.Start()
			benchConfig := config
			benchConfig.Benchmark = benchmark
			result, err := e.runBenchmark(benchConfig)
			if err != nil {
				benchmarkSuite.Fail(err)
				suiteStep.Fail(err)
				return err
			}
			benchmarkSuite.Complete()
			results = append(results, result)
		}
		suiteStep.Complete()
	}

	writer := new(tabwriter.Writer)
	writer.Init(os.Stdout, 0, 0, 3, ' ', tabwriter.FilterHTML)
	fmt.Fprintln(writer, "BENCHMARK\tREQUESTS\tDURATION\tTHROUGHPUT\tMEAN LATENCY\tMEDIAN LATENCY\t75% LATENCY\t95% LATENCY\t99% LATENCY")
	for _, result := range results {
		fmt.Fprintf(writer, "%s\t%d\t%s\t%f/sec\t%s\t%s\t%s\t%s\t%s\n",
			result.benchmark, result.requests, result.duration, result.throughput, result.meanLatency,
			result.latencyPercentiles[.5], result.latencyPercentiles[.75],
			result.latencyPercentiles[.95], result.latencyPercentiles[.99])
	}
	writer.Flush()
	return nil
}

// runBenchmark runs the given benchmark
func (e *suiteExecutor) runBenchmark() (result, error) {
	// Setup the benchmark
	if err := e.setupBenchmark(); err != nil {
		return result{}, err
	}

	wg := &sync.WaitGroup{}
	resultCh := make(chan *api.RunBenchmarkResponse, len(e.workers))
	errCh := make(chan error, len(e.workers))

	for _, worker := range e.workers {
		wg.Add(1)
		go func(worker api.BenchmarkerClient, iterations int, duration *time.Duration) {
			ctx := context.Background()
			for key, value := range e.config.Args {
				ctx = context.WithValue(ctx, key, value)
			}
			var durationProto *types.Duration
			if duration != nil {
				durationProto = types.DurationProto(*duration)
			}
			result, err := worker.RunBenchmark(context.Background(), &api.RunBenchmarkRequest{
				Suite:       e.config.Suite,
				Benchmark:   e.config.Benchmark,
				Iterations:  uint64(iterations),
				Duration:    durationProto,
				Parallelism: uint32(e.config.Parallelism),
			})
			if err != nil {
				errCh <- err
			} else {
				resultCh <- result
			}
			wg.Done()
		}(worker, e.config.Iterations/len(e.workers), e.config.Duration)
	}

	wg.Wait()
	close(resultCh)
	close(errCh)

	for err := range errCh {
		return result{}, err
	}

	var duration time.Duration
	var iterations uint64
	var meanLatencySum time.Duration
	var p50LatencySum time.Duration
	var p75LatencySum time.Duration
	var p95LatencySum time.Duration
	var p99LatencySum time.Duration
	for result := range resultCh {
		statistics := result.Statistics
		if statistics == nil {
			continue
		}
		iterations += statistics.Iterations
		if d, err := types.DurationFromProto(statistics.Duration); err == nil {
			duration = time.Duration(math.Max(float64(duration), float64(d)))
		} else {
			// TODO: log error
		}
		if d, err := types.DurationFromProto(statistics.MeanLatency); err == nil {
			meanLatencySum += d
		} else {
			// TODO: log error
		}
		if d, err := types.DurationFromProto(statistics.P50Latency); err == nil {
			p50LatencySum += d
		} else {
			// TODO: log error
		}
		if d, err := types.DurationFromProto(statistics.P75Latency); err == nil {
			p75LatencySum += d
		} else {
			// TODO: log error
		}
		if d, err := types.DurationFromProto(statistics.P95Latency); err == nil {
			p95LatencySum += d
		} else {
			// TODO: log error
		}
		if d, err := types.DurationFromProto(statistics.P99Latency); err == nil {
			p99LatencySum += d
		} else {
			// TODO: log error
		}
	}

	throughput := float64(iterations) / (float64(duration) / float64(time.Second))
	meanLatency := time.Duration(float64(meanLatencySum) / float64(len(e.workers)))
	latencyPercentiles := make(map[float32]time.Duration)
	latencyPercentiles[.5] = time.Duration(float64(p50LatencySum) / float64(len(e.workers)))
	latencyPercentiles[.75] = time.Duration(float64(p75LatencySum) / float64(len(e.workers)))
	latencyPercentiles[.95] = time.Duration(float64(p95LatencySum) / float64(len(e.workers)))
	latencyPercentiles[.99] = time.Duration(float64(p99LatencySum) / float64(len(e.workers)))

	return result{
		benchmark:          e.config.Benchmark,
		requests:           int(iterations),
		duration:           duration,
		throughput:         throughput,
		meanLatency:        meanLatency,
		latencyPercentiles: latencyPercentiles,
	}, nil
}

type result struct {
	benchmark          string
	requests           int
	duration           time.Duration
	throughput         float64
	meanLatency        time.Duration
	latencyPercentiles map[float32]time.Duration
}

// listBenchmarks returns a list of benchmarks in the given suite
func listBenchmarks(suite BenchmarkingSuite) []string {
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
