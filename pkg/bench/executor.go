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
	"github.com/onosproject/helmit/internal/log"
	"github.com/onosproject/helmit/internal/task"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"math"
	"path"
	"text/tabwriter"
	"time"
)

const managementPort = 5000

// newExecutor returns a new benchmark job
func newExecutor(spec job.Spec, writer log.Writer) (*benchExecutor, error) {
	return &benchExecutor{
		spec:   spec,
		jobs:   job.NewManager[WorkerConfig](job.WorkerType),
		writer: writer,
	}, nil
}

// benchExecutor coordinates workers for suites of benchmarks
type benchExecutor struct {
	spec   job.Spec
	jobs   *job.Manager[WorkerConfig]
	writer log.Writer
}

// Run runs the tests
func (e *benchExecutor) run(config Config) error {
	jobs := make(map[int]job.Job[WorkerConfig])
	for worker := 0; worker < config.Workers; worker++ {
		jobID := newWorkerName(e.spec.ID, worker)
		jobs[worker] = e.newJob(jobID, config)
	}

	err := task.New(e.writer, "Start benchmark workers").Run(func(context task.Context) error {
		var futures []task.Future
		for i := 0; i < config.Workers; i++ {
			futures = append(futures, func(worker int) task.Future {
				return context.NewTask("Start benchmark worker %d", worker).Start(func(context task.Context) error {
					context.Status().Setf("Creating worker %d", worker)
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

	workers := make(map[int]api.BenchmarkerClient)
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
		workers[i] = api.NewBenchmarkerClient(worker)
	}

	err = task.New(e.writer, "Setup suite").Run(func(context task.Context) error {
		var retErr error
		for worker, client := range workers {
			context.Status().Setf("Setting up worker %d", worker)
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

	err = task.New(e.writer, "Setup workers").Run(func(context task.Context) error {
		var futures []task.Future
		for i, client := range workers {
			futures = append(futures, func(client api.BenchmarkerClient) task.Future {
				return context.NewTask("Setup worker %d", i).Start(func(context task.Context) error {
					context.Status().Setf("Setting up worker %d", i)
					_, err := client.SetupBenchmarkWorker(e.newContext(config), &api.SetupBenchmarkWorkerRequest{
						Suite: config.Suite,
					})
					return err
				})
			}(client))
		}
		return task.Await(futures...)
	})
	if err != nil {
		return err
	}

	err = task.New(e.writer, "Start benchmark").Run(func(context task.Context) error {
		var futures []task.Future
		for i, client := range workers {
			futures = append(futures, func(client api.BenchmarkerClient) task.Future {
				return context.NewTask("Start worker %d", i).Start(func(context task.Context) error {
					context.Status().Setf("Starting worker %d", i)
					_, err := client.StartBenchmark(e.newContext(config), &api.StartBenchmarkRequest{
						Suite:       config.Suite,
						Benchmark:   config.Benchmark,
						Parallelism: uint32(config.Parallelism),
						Timeout:     types.DurationProto(e.spec.Timeout),
					})
					return err
				})
			}(client))
		}
		return task.Await(futures...)
	})
	if err != nil {
		return err
	}

	err = task.New(e.writer, "Run benchmark").Run(func(context task.Context) error {
		context.Status().Setf("Running benchmark %s", config.Benchmark)
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

		var report = func() error {
			writer := new(tabwriter.Writer)
			writer.Init(context.Writer(), 0, 0, 3, ' ', tabwriter.FilterHTML)
			fmt.Fprintln(writer, "\tREQUESTS\tDURATION\tTHROUGHPUT\tMEAN LATENCY\tMEDIAN LATENCY\t75% LATENCY\t95% LATENCY\t99% LATENCY")

			for worker, client := range workers {
				request := &api.ReportBenchmarkRequest{
					Suite:     config.Suite,
					Benchmark: config.Benchmark,
				}
				response, err := client.ReportBenchmark(e.newContext(config), request)
				if err != nil {
					return err
				} else {
					report := response.Report
					iterations += report.Iterations
					duration, err := types.DurationFromProto(report.Duration)
					if err != nil {
						return err
					}
					maxDuration = time.Duration(math.Max(float64(maxDuration), float64(duration)))

					meanLatency, err := types.DurationFromProto(report.MeanLatency)
					if err != nil {
						return err
					}
					meanLatencySum += meanLatency

					p50Latency, err := types.DurationFromProto(report.P50Latency)
					if err != nil {
						return err
					}
					p50LatencySum += p50Latency

					p75Latency, err := types.DurationFromProto(report.P75Latency)
					if err != nil {
						return err
					}
					p75LatencySum += p75Latency

					p95Latency, err := types.DurationFromProto(report.P95Latency)
					if err != nil {
						return err
					}
					p95LatencySum += p95Latency

					p99Latency, err := types.DurationFromProto(report.P99Latency)
					if err != nil {
						return err
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
			return nil
		}

		if config.Duration != nil {
			ticker := time.NewTicker(config.ReportInterval)
			timer := time.NewTimer(*config.Duration)
			for {
				select {
				case <-ticker.C:
					if err := report(); err != nil {
						return err
					}
				case <-timer.C:
					break
				}
			}
		} else {
			ticker := time.NewTicker(config.ReportInterval)
			for range ticker.C {
				if err := report(); err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	err = task.New(e.writer, "Stop benchmark").Run(func(context task.Context) error {
		var futures []task.Future
		for i, client := range workers {
			futures = append(futures, func(client api.BenchmarkerClient) task.Future {
				return context.NewTask("Stop worker %d", i).Start(func(context task.Context) error {
					context.Status().Setf("Stopping worker %d", i)
					_, err := client.StopBenchmark(e.newContext(config), &api.StopBenchmarkRequest{
						Suite:     config.Suite,
						Benchmark: config.Benchmark,
					})
					return err
				})
			}(client))
		}
		return task.Await(futures...)
	})
	if err != nil {
		return err
	}

	err = task.New(e.writer, "Tear down workers").Run(func(context task.Context) error {
		var futures []task.Future
		for i, client := range workers {
			futures = append(futures, func(client api.BenchmarkerClient) task.Future {
				return context.NewTask("Tear down worker %d", i).Start(func(context task.Context) error {
					context.Status().Setf("Tearing down worker %d", i)
					_, err := client.TearDownBenchmarkWorker(e.newContext(config), &api.TearDownBenchmarkWorkerRequest{
						Suite: config.Suite,
					})
					return err
				})
			}(client))
		}
		return task.Await(futures...)
	})
	if err != nil {
		return err
	}

	err = task.New(e.writer, "Stop workers").Run(func(context task.Context) error {
		var futures []task.Future
		for i := 0; i < config.Workers; i++ {
			futures = append(futures, func(worker int) task.Future {
				return context.NewTask("Stop worker %d", worker).Start(func(context task.Context) error {
					if err := e.jobs.Stop(jobs[worker], context.Status()); err != nil {
						return err
					}
					return nil
				})
			}(i))
		}
		return task.Await(futures...)
	})
	return nil
}

func newWorkerName(jobID string, worker int) string {
	return fmt.Sprintf("%s-worker-%d", jobID, worker)
}

func (e *benchExecutor) getWorkerAddress(worker int) string {
	return fmt.Sprintf("%s:%d", newWorkerName(e.spec.ID, worker), managementPort)
}

func (e *benchExecutor) newContext(config Config) context.Context {
	ctx := context.Background()
	for key, value := range config.Args {
		ctx = context.WithValue(ctx, key, value)
	}
	return ctx
}

func (e *benchExecutor) newJob(id string, config Config) job.Job[WorkerConfig] {
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
