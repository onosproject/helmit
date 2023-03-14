// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package benchmark

import (
	"context"
	"fmt"
	"math"
	"os"
	"sync"
	"text/tabwriter"
	"time"

	grpc_retry "github.com/grpc-ecosystem/go-grpc-middleware/retry"

	"github.com/onosproject/helmit/pkg/job"
	"github.com/onosproject/helmit/pkg/util/async"
	"github.com/onosproject/helmit/pkg/util/logging"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

// newCoordinator returns a new benchmark coordinator
func newCoordinator(suites map[string]BenchmarkingSuite, config *Config) (*Coordinator, error) {
	return &Coordinator{
		suites: suites,
		config: config,
		runner: job.NewNamespace(config.Namespace),
	}, nil
}

// Coordinator coordinates workers for suites of benchmarks
type Coordinator struct {
	suites map[string]BenchmarkingSuite
	config *Config
	runner *job.Runner
}

// Run runs the tests
func (c *Coordinator) Run() (int, error) {
	var suites []string
	if c.config.Suite == "" {
		for name := range c.suites {
			suites = append(suites, name)
		}
	} else {
		suites = []string{c.config.Suite}
	}

	var returnCode int
	for _, suite := range suites {
		jobID := newJobID(c.config.ID, suite)
		config := &Config{
			Config: &job.Config{
				ID:              jobID,
				Namespace:       c.config.Config.Namespace,
				ServiceAccount:  c.config.Config.ServiceAccount,
				Labels:          c.config.Config.Labels,
				Annotations:     c.config.Config.Annotations,
				Image:           c.config.Config.Image,
				ImagePullPolicy: c.config.Config.ImagePullPolicy,
				Executable:      c.config.Config.Executable,
				Context:         c.config.Config.Context,
				Values:          c.config.Config.Values,
				ValueFiles:      c.config.Config.ValueFiles,
				Env:             c.config.Config.Env,
				Timeout:         c.config.Config.Timeout,
				NoTeardown:      c.config.Config.NoTeardown,
				Secrets:         c.config.Config.Secrets,
			},
			Suite:       suite,
			Benchmark:   c.config.Benchmark,
			Workers:     c.config.Workers,
			Parallelism: c.config.Parallelism,
			Iterations:  c.config.Iterations,
			Duration:    c.config.Duration,
			MaxLatency:  c.config.MaxLatency,
			Args:        c.config.Args,
			NoTeardown:  c.config.Config.NoTeardown,
		}
		task := &WorkerTask{
			suites: c.suites,
			runner: c.runner,
			config: config,
		}
		status, err := task.Run()
		if err != nil {
			return status, err
		} else if returnCode == 0 {
			returnCode = status
		}
	}
	return returnCode, nil
}

// newJobID returns a new unique test job ID
func newJobID(testID, suite string) string {
	return fmt.Sprintf("%s-%s", testID, suite)
}

// WorkerTask manages a single test job for a test worker
type WorkerTask struct {
	suites  map[string]BenchmarkingSuite
	runner  *job.Runner
	config  *Config
	workers []WorkerServiceClient
}

// Run runs the worker job
func (t *WorkerTask) Run() (int, error) {
	// Start the job
	err := t.run()
	return 0, err
}

// start starts the test job
func (t *WorkerTask) run() error {
	if err := t.createWorkers(); err != nil {
		return err
	}
	if err := t.runBenchmarks(); err != nil {
		return err
	}
	return nil
}

func getWorkerName(worker int, jobID string) string {
	return fmt.Sprintf("%s-worker-%d", jobID, worker)
}

func (t *WorkerTask) getWorkerAddress(worker int) string {
	return fmt.Sprintf("%s:5000", getWorkerName(worker, t.config.ID))
}

// createWorkers creates the benchmark workers
func (t *WorkerTask) createWorkers() error {
	return async.IterAsync(t.config.Workers, t.createWorker)
}

// createWorker creates the given worker
func (t *WorkerTask) createWorker(worker int) error {
	jobID := getWorkerName(worker, t.config.ID)
	env := t.config.Env
	if env == nil {
		env = make(map[string]string)
	}
	env[benchmarkTypeEnv] = string(benchmarkTypeWorker)
	env[benchmarkWorkerEnv] = fmt.Sprintf("%d", worker)
	env[benchmarkJobEnv] = t.config.ID
	job := &job.Job{
		Config: &job.Config{
			ID:              jobID,
			Namespace:       t.config.Config.Namespace,
			ServiceAccount:  t.config.Config.ServiceAccount,
			Labels:          t.config.Config.Labels,
			Annotations:     t.config.Config.Annotations,
			Image:           t.config.Config.Image,
			ImagePullPolicy: t.config.Config.ImagePullPolicy,
			Executable:      t.config.Config.Executable,
			Context:         t.config.Config.Context,
			Values:          t.config.Config.Values,
			ValueFiles:      t.config.Config.ValueFiles,
			Env:             env,
			Timeout:         t.config.Config.Timeout,
			NoTeardown:      t.config.Config.NoTeardown,
			Secrets:         t.config.Config.Secrets,
		},
		JobConfig: &Config{
			Config: &job.Config{
				ID:              jobID,
				Namespace:       t.config.Config.Namespace,
				ServiceAccount:  t.config.Config.ServiceAccount,
				Labels:          t.config.Config.Labels,
				Annotations:     t.config.Config.Annotations,
				Image:           t.config.Config.Image,
				ImagePullPolicy: t.config.Config.ImagePullPolicy,
				Executable:      t.config.Config.Executable,
				Context:         t.config.Config.Context,
				Values:          t.config.Config.Values,
				ValueFiles:      t.config.Config.ValueFiles,
				Env:             env,
				Timeout:         t.config.Config.Timeout,
				NoTeardown:      t.config.Config.NoTeardown,
				Secrets:         t.config.Config.Secrets,
			},
			Suite:       t.config.Suite,
			Benchmark:   t.config.Benchmark,
			Workers:     t.config.Workers,
			Parallelism: t.config.Parallelism,
			Iterations:  t.config.Iterations,
			Duration:    t.config.Duration,
			MaxLatency:  t.config.MaxLatency,
			Args:        t.config.Args,
			NoTeardown:  t.config.Config.NoTeardown,
		},
		Type: benchmarkJobType,
	}
	return t.runner.StartJob(job)
}

// getWorkerConns returns the worker clients for the given benchmark
func (t *WorkerTask) getWorkers() ([]WorkerServiceClient, error) {
	if t.workers != nil {
		return t.workers, nil
	}

	workers := make([]WorkerServiceClient, t.config.Workers)
	for i := 0; i < t.config.Workers; i++ {
		worker, err := grpc.Dial(
			t.getWorkerAddress(i),
			grpc.WithInsecure(),
			grpc.WithUnaryInterceptor(
				grpc_retry.UnaryClientInterceptor(
					grpc_retry.WithCodes(codes.Unavailable),
					grpc_retry.WithBackoff(grpc_retry.BackoffExponential(1*time.Second)),
					grpc_retry.WithMax(10))))
		if err != nil {
			return nil, err
		}
		workers[i] = NewWorkerServiceClient(worker)
	}
	t.workers = workers
	return workers, nil
}

// setupSuite sets up the benchmark suite
func (t *WorkerTask) setupSuite() error {
	workers, err := t.getWorkers()
	if err != nil {
		return err
	}

	worker := workers[0]
	_, err = worker.SetupSuite(context.Background(), &SuiteRequest{
		Suite: t.config.Suite,
		Args:  t.config.Args,
	})
	return err
}

// setupWorkers sets up the benchmark workers
func (t *WorkerTask) setupWorkers() error {
	workers, err := t.getWorkers()
	if err != nil {
		return err
	}

	wg := &sync.WaitGroup{}
	errCh := make(chan error)
	for _, worker := range workers {
		wg.Add(1)
		go func(worker WorkerServiceClient) {
			_, err = worker.SetupWorker(context.Background(), &SuiteRequest{
				Suite: t.config.Suite,
				Args:  t.config.Args,
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
func (t *WorkerTask) setupBenchmark(benchmark string) error {
	workers, err := t.getWorkers()
	if err != nil {
		return err
	}

	wg := &sync.WaitGroup{}
	errCh := make(chan error)
	for _, worker := range workers {
		wg.Add(1)
		go func(worker WorkerServiceClient) {
			_, err = worker.SetupBenchmark(context.Background(), &BenchmarkRequest{
				Suite:     t.config.Suite,
				Benchmark: benchmark,
				Args:      t.config.Args,
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
func (t *WorkerTask) runBenchmarks() error {
	// Setup the benchmark suite on one of the workers
	if err := t.setupSuite(); err != nil {
		return err
	}

	// Setup the workers
	if err := t.setupWorkers(); err != nil {
		return err
	}

	// Run the benchmarks
	results := make([]result, 0)
	if t.config.Benchmark != "" {
		step := logging.NewStep(t.config.ID, "Run benchmark %s", t.config.Benchmark)
		step.Start()
		result, err := t.runBenchmark(t.config.Benchmark)
		if err != nil {
			step.Fail(err)
			return err
		}
		step.Complete()
		results = append(results, result)
	} else {
		suiteStep := logging.NewStep(t.config.ID, "Run benchmark suite %s", t.config.Suite)
		suiteStep.Start()
		suite := t.suites[t.config.Suite]
		benchmarks := getBenchmarks(suite)
		for _, benchmark := range benchmarks {
			benchmarkSuite := logging.NewStep(t.config.ID, "Run benchmark %s", benchmark)
			benchmarkSuite.Start()
			result, err := t.runBenchmark(benchmark)
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

	for _, result := range results {
		if t.config.MaxLatency != nil && result.meanLatency >= *t.config.MaxLatency {
			return fmt.Errorf("mean latency of %d exceeds maximum of %v", result.meanLatency.Milliseconds(), t.config.MaxLatency)
		}
	}
	return nil
}

// runBenchmark runs the given benchmark
func (t *WorkerTask) runBenchmark(benchmark string) (result, error) {
	// Setup the benchmark
	if err := t.setupBenchmark(benchmark); err != nil {
		return result{}, err
	}

	workers, err := t.getWorkers()
	if err != nil {
		return result{}, err
	}

	wg := &sync.WaitGroup{}
	resultCh := make(chan *RunResponse, len(workers))
	errCh := make(chan error, len(workers))

	for _, worker := range workers {
		wg.Add(1)
		go func(worker WorkerServiceClient, requests int, duration *time.Duration) {
			result, err := worker.RunBenchmark(context.Background(), &RunRequest{
				Suite:       t.config.Suite,
				Benchmark:   benchmark,
				Requests:    uint32(requests),
				Duration:    duration,
				MaxLatency:  t.config.MaxLatency,
				Parallelism: uint32(t.config.Parallelism),
				Args:        t.config.Args,
			})
			if err != nil {
				errCh <- err
			} else {
				resultCh <- result
			}
			wg.Done()
		}(worker, t.config.Iterations/len(workers), t.config.Duration)
	}

	wg.Wait()
	close(resultCh)
	close(errCh)

	for err := range errCh {
		return result{}, err
	}

	var duration time.Duration
	var requests uint32
	var latencySum time.Duration
	var latency50Sum time.Duration
	var latency75Sum time.Duration
	var latency95Sum time.Duration
	var latency99Sum time.Duration
	for result := range resultCh {
		requests += result.Requests
		duration = time.Duration(math.Max(float64(duration), float64(result.Duration)))
		latencySum += result.Latency
		latency50Sum += result.Latency50
		latency75Sum += result.Latency75
		latency95Sum += result.Latency95
		latency99Sum += result.Latency99
	}

	throughput := float64(requests) / (float64(duration) / float64(time.Second))
	meanLatency := time.Duration(float64(latencySum) / float64(len(workers)))
	latencyPercentiles := make(map[float32]time.Duration)
	latencyPercentiles[.5] = time.Duration(float64(latency50Sum) / float64(len(workers)))
	latencyPercentiles[.75] = time.Duration(float64(latency75Sum) / float64(len(workers)))
	latencyPercentiles[.95] = time.Duration(float64(latency95Sum) / float64(len(workers)))
	latencyPercentiles[.99] = time.Duration(float64(latency99Sum) / float64(len(workers)))

	return result{
		benchmark:          benchmark,
		requests:           int(requests),
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
