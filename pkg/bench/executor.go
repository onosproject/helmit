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
	"github.com/onosproject/helmit/internal/console"
	"github.com/onosproject/helmit/internal/job"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"math"
	"text/tabwriter"
	"time"
)

// newExecutor returns a new benchmark job
func newExecutor(spec job.Spec) (*benchExecutor, error) {
	return &benchExecutor{
		spec: spec,
		jobs: job.NewManager[WorkerConfig](),
	}, nil
}

// benchExecutor coordinates workers for suites of benchmarks
type benchExecutor struct {
	spec job.Spec
	jobs *job.Manager[WorkerConfig]
}

// Run runs the tests
func (e *benchExecutor) run(config Config, context *console.Context) error {
	err := context.Run("Deploying workers", func(task *console.Task) error {
		var waiters []console.Waiter
		for worker := 0; worker < config.Workers; worker++ {
			waiters = append(waiters, task.RunAsync(fmt.Sprintf("Deploying worker %d", worker), func(task *console.Task) error {
				return e.createWorker(config, worker, task)
			}))
		}
		return console.Wait(waiters...)
	})
	if err != nil {
		return err
	}

	workers := make(map[int]api.BenchmarkerClient)
	for i := 0; i < config.Workers; i++ {
		worker, err := grpc.Dial(
			fmt.Sprintf("%s:5000", newWorkerName(e.spec.ID, config.Suite, i)),
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithUnaryInterceptor(
				grpc_retry.UnaryClientInterceptor(
					grpc_retry.WithCodes(codes.Unavailable),
					grpc_retry.WithBackoff(grpc_retry.BackoffExponential(1*time.Second)),
					grpc_retry.WithMax(10))))
		if err != nil {
			return err
		}
		workers[i] = api.NewBenchmarkerClient(worker)
	}

	err = context.Run("Setting up benchmark suite", func(task *console.Task) error {
		var retErr error
		for _, client := range workers {
			_, err := client.SetupBenchmarkSuite(e.newContext(config), &api.SetupBenchmarkSuiteRequest{
				Suite: config.Suite,
			})
			if err != nil {
				retErr = err
			} else {
				retErr = nil
				break
			}
		}
		return retErr
	})
	if err != nil {
		return err
	}

	err = context.Run("Setting up workers", func(task *console.Task) error {
		var waiters []console.Waiter
		for i, client := range workers {
			waiters = append(waiters, func(client api.BenchmarkerClient) console.Waiter {
				return task.RunAsync(fmt.Sprintf("Setting up worker %d", i), func(task *console.Task) error {
					_, err := client.SetupBenchmarkWorker(e.newContext(config), &api.SetupBenchmarkWorkerRequest{
						Suite: config.Suite,
					})
					return err
				})
			}(client))
		}
		return console.Wait(waiters...)
	})
	if err != nil {
		return err
	}

	err = context.Run("Starting benchmark", func(task *console.Task) error {
		var waiters []console.Waiter
		for worker, client := range workers {
			waiters = append(waiters, func(worker int, client api.BenchmarkerClient) console.Waiter {
				return task.RunAsync(fmt.Sprintf("Starting worker %d", worker), func(task *console.Task) error {
					_, err := client.StartBenchmark(e.newContext(config), &api.StartBenchmarkRequest{
						Suite:       config.Suite,
						Benchmark:   config.Benchmark,
						Parallelism: uint32(config.Parallelism),
						Timeout:     types.DurationProto(e.spec.Timeout),
					})
					return err
				})
			}(worker, client))
		}
		return console.Wait(waiters...)
	})
	if err != nil {
		return err
	}

	err = context.Run("Running benchmark", func(task *console.Task) error {
		err := async.IterAsync(len(workers), func(i int) error {
			client := workers[i]
			request := &api.StartBenchmarkRequest{
				Suite:       config.Suite,
				Benchmark:   config.Benchmark,
				Parallelism: uint32(config.Parallelism),
			}
			_, err := client.StartBenchmark(e.newContext(config), request)
			return err
		})
		if err != nil {
			return err
		}

		var maxDuration time.Duration
		var iterations uint64
		var meanLatencySum time.Duration
		var p50LatencySum time.Duration
		var p75LatencySum time.Duration
		var p95LatencySum time.Duration
		var p99LatencySum time.Duration

		var report = func() {
			writer := new(tabwriter.Writer)
			writer.Init(task.Writer(), 0, 0, 3, ' ', tabwriter.FilterHTML)
			fmt.Fprintln(writer, "\tREQUESTS\tDURATION\tTHROUGHPUT\tMEAN LATENCY\tMEDIAN LATENCY\t75% LATENCY\t95% LATENCY\t99% LATENCY")

			for worker, client := range workers {
				request := &api.ReportBenchmarkRequest{
					Suite:     config.Suite,
					Benchmark: config.Benchmark,
				}
				response, err := client.ReportBenchmark(e.newContext(config), request)
				if err != nil {
					task.Log(err.Error())
					continue
				} else {
					report := response.Report
					iterations += report.Iterations
					duration, err := types.DurationFromProto(report.Duration)
					if err != nil {
						task.Log(err.Error())
						continue
					}
					maxDuration = time.Duration(math.Max(float64(maxDuration), float64(duration)))

					meanLatency, err := types.DurationFromProto(report.MeanLatency)
					if err != nil {
						task.Log(err.Error())
						continue
					}
					meanLatencySum += meanLatency

					p50Latency, err := types.DurationFromProto(report.P50Latency)
					if err != nil {
						task.Log(err.Error())
						continue
					}
					p50LatencySum += p50Latency

					p75Latency, err := types.DurationFromProto(report.P75Latency)
					if err != nil {
						task.Log(err.Error())
						continue
					}
					p75LatencySum += p75Latency

					p95Latency, err := types.DurationFromProto(report.P95Latency)
					if err != nil {
						task.Log(err.Error())
						continue
					}
					p95LatencySum += p95Latency

					p99Latency, err := types.DurationFromProto(report.P99Latency)
					if err != nil {
						task.Log(err.Error())
						continue
					}
					p99LatencySum += p99Latency

					throughput := float64(report.Iterations) / (float64(duration) / float64(time.Second))
					fmt.Fprintf(writer, "WORKER %d\t%d\t%s\t%f/sec\t%s\t%s\t%s\t%s\t%s\n",
						worker, report.Iterations, report.Duration, throughput,
						meanLatency, p50Latency, p75Latency, p95Latency, p99Latency)
				}
			}

			throughput := float64(iterations) / (float64(maxDuration) / float64(time.Second))
			meanLatency := time.Duration(float64(meanLatencySum) / float64(len(workers)))
			p50Latency := time.Duration(float64(p50LatencySum) / float64(len(workers)))
			p75Latency := time.Duration(float64(p75LatencySum) / float64(len(workers)))
			p95Latency := time.Duration(float64(p95LatencySum) / float64(len(workers)))
			p99Latency := time.Duration(float64(p99LatencySum) / float64(len(workers)))

			fmt.Fprintf(writer, "total\t%d\t%s\t%f/sec\t%s\t%s\t%s\t%s\t%s\n",
				iterations, maxDuration, throughput, meanLatency,
				p50Latency, p75Latency, p95Latency, p99Latency)

			writer.Flush()
		}

		if config.Duration != nil {
			ticker := time.NewTicker(config.ReportInterval)
			timer := time.NewTimer(*config.Duration)
			for {
				select {
				case <-ticker.C:
					report()
				case <-timer.C:
					break
				}
			}
		} else {
			ticker := time.NewTicker(config.ReportInterval)
			for range ticker.C {
				report()
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	err = context.Run("Stopping benchmark", func(task *console.Task) error {
		var waiters []console.Waiter
		for worker, client := range workers {
			waiters = append(waiters, func(worker int, client api.BenchmarkerClient) console.Waiter {
				return task.RunAsync(fmt.Sprintf("Stopping worker %d", worker), func(task *console.Task) error {
					_, err := client.StopBenchmark(e.newContext(config), &api.StopBenchmarkRequest{
						Suite:     config.Suite,
						Benchmark: config.Benchmark,
					})
					return err
				})
			}(worker, client))
		}
		return console.Wait(waiters...)
	})
	if err != nil {
		return err
	}

	err = context.Run("Tearing down workers", func(task *console.Task) error {
		var waiters []console.Waiter
		for i, client := range workers {
			waiters = append(waiters, func(client api.BenchmarkerClient) console.Waiter {
				return task.RunAsync(fmt.Sprintf("Tearing down worker %d", i), func(task *console.Task) error {
					_, err := client.TearDownBenchmarkWorker(e.newContext(config), &api.TearDownBenchmarkWorkerRequest{
						Suite: config.Suite,
					})
					return err
				})
			}(client))
		}
		return console.Wait(waiters...)
	})
	if err != nil {
		return err
	}

	err = context.Run("Tearing down benchmark suite", func(task *console.Task) error {
		var retErr error
		for _, client := range workers {
			_, err := client.TearDownBenchmarkSuite(e.newContext(config), &api.TearDownBenchmarkSuiteRequest{
				Suite: config.Suite,
			})
			if err != nil {
				retErr = err
			} else {
				retErr = nil
				break
			}
		}
		return retErr
	})
	if err != nil {
		return err
	}
	return nil
}

func newWorkerName(jobID string, suite string, worker int) string {
	return fmt.Sprintf("%s-%s-worker-%d", jobID, suite, worker)
}

func (e *benchExecutor) getWorkerAddress(config Config, worker int) string {
	return fmt.Sprintf("%s:5000", newWorkerName(e.spec.ID, config.Suite, worker))
}

// createWorker creates the given worker
func (e *benchExecutor) createWorker(config Config, worker int, task *console.Task) error {
	jobID := newWorkerName(e.spec.ID, config.Suite, worker)
	env := config.WorkerConfig.Env
	if env == nil {
		env = make(map[string]string)
	}
	env[benchmarkTypeEnv] = string(benchTypeWorker)
	env[benchmarkWorkerEnv] = fmt.Sprintf("%d", worker)
	return e.jobs.Start(job.Job[WorkerConfig]{
		Spec: job.Spec{
			ID:              jobID,
			Namespace:       e.spec.Namespace,
			ServiceAccount:  e.spec.ServiceAccount,
			Labels:          e.spec.Labels,
			Annotations:     e.spec.Annotations,
			Image:           config.WorkerConfig.Image,
			ImagePullPolicy: config.WorkerConfig.ImagePullPolicy,
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
	}, task)
}

func (e *benchExecutor) newContext(config Config) context.Context {
	ctx := context.Background()
	for key, value := range config.Args {
		ctx = context.WithValue(ctx, key, value)
	}
	return ctx
}
