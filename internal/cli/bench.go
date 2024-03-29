// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	petname "github.com/dustinkirkland/golang-petname"
	"github.com/gosuri/uilive"
	"github.com/onosproject/helmit/internal/build"
	"github.com/onosproject/helmit/internal/logging"
	"github.com/onosproject/helmit/pkg/benchmark"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"text/tabwriter"
	"time"

	"github.com/onosproject/helmit/internal/job"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
)

const benchExamples = `
  # Run benchmarks packaged in a Docker image.
  helmit bench --image atomix/kubernetes-benchmarks:latest --duration 1m

  # Run benchmarks by referencing a command package and providing a context.
  # The specified context will be loaded into the benchmark pods as the current working directory.
  helmit bench ./cmd/benchmarks --context ./charts --iterations 1000

  # Run benchmarks in a specific namespace.
  helmit bench ./cmd/benchmarks -n bench --suite atomix --duration 5m

  # Run a benchmark suite by name.
  helmit bench ./cmd/benchmarks -c ./charts --suite atomix --duration 5m

  # Run a single benchmark function by name.
  helmit bench ./cmd/benchmarks -c ./charts --suite atomix --benchmark BenchmarkGet --duration 5m

  # Parallelize benchmark clients across goroutines.
  helmit bench ./cmd/benchmarks -c ./charts --suite atomix --parallel 10 --duration 1m

  # Parallelize benchmark clients across worker pods.
  helmit bench ./cmd/benchmarks -c ./charts --suite atomix --workers 4 --duration 1m

  # Override Helm chart values with flags.
  # Value overrids must be namespaced with the name of the release to which to apply the value.
  helmit bench ./cmd/benchmarks -c ./charts --set atomix-controller.image=atomix/atomix-controller:latest --set atomix-raft.replicas=3 --suite atomix --iterations 1000

  # Override Helm chart values with values files.
  # Values files must be key/value pairs where the key is the Helm release name and the value the path to the file.
  helmit bench ./cmd/benchmarks -c ./charts -f atomix-controller=./atomix-controller.yaml --suite atomix --duration 1m
`

const shutdownFile = "/tmp/shutdown"

func getBenchCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "bench",
		Aliases: []string{"benchmark", "benchmarks"},
		Short:   "Run benchmarks on Kubernetes",
		Example: benchExamples,
		Args:    cobra.ArbitraryArgs,
		RunE:    runBenchCommand,
	}
	cmd.Flags().StringP("namespace", "n", "", "the namespace in which to run the benchmarks")
	cmd.Flags().Bool("create-namespace", false, "whether to create the namespace when running the test")
	cmd.Flags().String("service-account", "", "the name of the service account to use to run worker pods")
	cmd.Flags().StringToStringP("label", "l", map[string]string{}, "labels to apply to the worker pods")
	cmd.Flags().StringToStringP("annotation", "a", map[string]string{}, "annotations to apply to the worker pods")
	cmd.Flags().StringP("context", "c", "", "the benchmark context")
	cmd.Flags().StringP("image", "i", "", "the benchmark image to run")
	cmd.Flags().String("image-pull-policy", string(corev1.PullIfNotPresent), "the Docker image pull policy")
	cmd.Flags().StringArrayP("values", "f", []string{}, "release values paths")
	cmd.Flags().StringArray("set", []string{}, "cluster argument overrides")
	cmd.Flags().StringP("suite", "s", "", "the benchmark suite to run")
	cmd.Flags().StringP("benchmark", "b", "BenchmarkSuite$", "the name of the benchmark to run")
	cmd.Flags().IntP("workers", "w", 1, "the number of workers to run")
	cmd.Flags().Int("parallel", 1, "the number of concurrent goroutines per client")
	cmd.Flags().IntP("iterations", "", 0, "the number of iterations to run")
	cmd.Flags().DurationP("duration", "d", 0, "the duration for which to run the test")
	cmd.Flags().DurationP("report-interval", "r", 5*time.Second, "the interval at which to report benchmark results")
	cmd.Flags().StringToString("arg", map[string]string{}, "a mapping of named benchmark arguments")
	cmd.Flags().Duration("timeout", 10*time.Minute, "benchmark timeout")
	cmd.Flags().Bool("no-teardown", false, "do not tear down clusters following benchmarks")
	cmd.Flags().StringSlice("secret", []string{}, "secrets to pass to the kubernetes pod")
	_ = cmd.MarkFlagRequired("suite")
	_ = cmd.MarkFlagRequired("benchmark")
	return cmd
}

func runBenchCommand(cmd *cobra.Command, args []string) error {
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true

	namespace, _ := cmd.Flags().GetString("namespace")
	createNamespace, _ := cmd.Flags().GetBool("create-namespace")
	serviceAccount, _ := cmd.Flags().GetString("service-account")
	labels, _ := cmd.Flags().GetStringToString("label")
	annotations, _ := cmd.Flags().GetStringToString("annotation")
	contextPath, _ := cmd.Flags().GetString("context")
	image, _ := cmd.Flags().GetString("image")
	suite, _ := cmd.Flags().GetString("suite")
	benchmarkName, _ := cmd.Flags().GetString("benchmark")
	workers, _ := cmd.Flags().GetInt("workers")
	parallelism, _ := cmd.Flags().GetInt("parallel")
	iterations, _ := cmd.Flags().GetInt("iterations")
	duration, _ := cmd.Flags().GetDuration("duration")
	reportInterval, _ := cmd.Flags().GetDuration("report-interval")
	files, _ := cmd.Flags().GetStringArray("values")
	sets, _ := cmd.Flags().GetStringArray("set")
	benchArgs, _ := cmd.Flags().GetStringToString("args")
	timeout, _ := cmd.Flags().GetDuration("timeout")
	imagePullPolicy, _ := cmd.Flags().GetString("image-pull-policy")
	pullPolicy := corev1.PullPolicy(imagePullPolicy)
	noTeardown, _ := cmd.Flags().GetBool("no-teardown")
	secretsArray, _ := cmd.Flags().GetStringSlice("secret")

	// Either a command package or image must be specified
	pkgPaths := args
	if len(pkgPaths) == 0 && image == "" {
		return errors.New("must specify either a benchmark package or --image to run")
	}

	// Generate a unique benchmark ID
	benchID := petname.Generate(2, "-")

	// If the create-namespace is enabled, generate a default namespace if not specified.
	if namespace == "" {
		if createNamespace {
			namespace = benchID
		} else {
			namespace = "default"
		}
	}

	// If a context was provided, convert the context to its absolute path
	if contextPath != "" {
		path, err := filepath.Abs(contextPath)
		if err != nil {
			return err
		}
		contextPath = path
	}

	valueFiles, err := parseFiles(files)
	if err != nil {
		return err
	}

	values, err := parseOverrides(sets)
	if err != nil {
		return err
	}

	secrets, err := parseSecrets(secretsArray)
	if err != nil {
		return err
	}

	var executable string
	if len(pkgPaths) > 0 {
		step := logging.NewStep(benchID, "Preparing artifacts")
		step.Start()
		executable = filepath.Join(os.TempDir(), "helmit", benchID)
		defer os.RemoveAll(executable)
		image = defaultRunnerImage
		if err := build.Benchmarks(step, suite).Build(executable, pkgPaths...); err != nil {
			step.Fail(err)
			return err
		}
		step.Complete()
	}

	config := benchmark.Config{
		Namespace:      namespace,
		Suite:          suite,
		Benchmark:      benchmarkName,
		Parallelism:    parallelism,
		Values:         values,
		ReportInterval: reportInterval,
		Timeout:        timeout,
		Args:           benchArgs,
		NoTeardown:     noTeardown,
	}

	if contextPath != "" {
		config.Context = filepath.Join(job.HomeDir, job.ContextDir)
	}

	if len(valueFiles) > 0 {
		config.ValueFiles = make(map[string][]string)
		for release, releaseFiles := range valueFiles {
			var absFiles []string
			for _, releaseFile := range releaseFiles {
				absFiles = append(absFiles, filepath.Join(job.HomeDir, filepath.Base(releaseFile)))
			}
			config.ValueFiles[release] = absFiles
		}
	}

	job := job.Job[benchmark.Config]{
		ID:              benchID,
		Namespace:       namespace,
		Labels:          labels,
		Annotations:     annotations,
		CreateNamespace: createNamespace,
		DeleteNamespace: createNamespace && !noTeardown,
		ServiceAccount:  serviceAccount,
		Image:           image,
		ImagePullPolicy: pullPolicy,
		Executable:      executable,
		Context:         contextPath,
		ValueFiles:      valueFiles,
		Secrets:         secrets,
		Config:          config,
	}

	if err := setupBenchmark(job, timeout); err != nil {
		return err
	}
	if err := runBenchmark(job, workers, iterations, duration, timeout); err != nil {
		return err
	}
	if err := tearDownBenchmark(job, timeout); err != nil {
		return err
	}
	return nil
}

func runJob(ctx context.Context, job job.Job[benchmark.Config], log logging.Logger) error {
	if err := job.Create(ctx, log); err != nil {
		return err
	}

	stream, err := job.GetLogs(ctx)
	if err != nil {
		return err
	}
	defer stream.Close()

	scanner := bufio.NewScanner(stream)
	for scanner.Scan() {
		fmt.Fprintf(os.Stdout, "    %s\n", scanner.Text())
	}

	if err := job.Delete(ctx, log); err != nil {
		return err
	}
	return nil
}

func setupBenchmark(job job.Job[benchmark.Config], timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	job.Config.Type = benchmark.SetupType
	job.DeleteNamespace = false
	step := logging.NewStep(job.ID, "Setting up benchmark")
	step.Start()
	if err := runJob(ctx, job, step); err != nil {
		step.Fail(err)
		return err
	}
	step.Complete()
	return nil
}

func runBenchmark(job job.Job[benchmark.Config], workers int, maxIterations int, maxDuration time.Duration, timeout time.Duration) error {
	ctx, cancel := context.WithCancel(context.Background())
	if maxDuration > 0 {
		ctx, cancel = context.WithTimeout(ctx, maxDuration)
	}
	defer cancel()

	reportCh := make(chan workerReport)
	wg := &sync.WaitGroup{}
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(worker int) {
			_ = runBenchmarkWorker(ctx, job, worker, reportCh, timeout)
			wg.Done()
		}(i)
	}

	go func() {
		wg.Wait()
		close(reportCh)
	}()

	uiwriter := uilive.New()
	uiwriter.Out = os.Stdout

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)

	reports := make([]*workerReport, workers)
	var canceled bool
	var iterations int
	for {
		select {
		case report, ok := <-reportCh:
			if !ok {
				return nil
			}
			if canceled {
				continue
			}

			reports[report.worker] = &report

			writer := new(tabwriter.Writer)
			writer.Init(uiwriter, 0, 0, 3, ' ', tabwriter.FilterHTML)

			fmt.Fprintln(writer, "WORKER\tITERATIONS\tDURATION\tTHROUGHPUT\tMEAN LATENCY\tMEDIAN LATENCY\t75% LATENCY\t95% LATENCY\t99% LATENCY")
			var count int
			var total benchmark.Report
			for worker, report := range reports {
				if report != nil {
					fmt.Fprintf(writer, "%d\t%d\t%s\t%f/sec\t%s\t%s\t%s\t%s\t%s\n",
						worker, report.Iterations, report.Duration,
						float64(report.Iterations)/(float64(report.Duration)/float64(time.Second)),
						report.MeanLatency, report.P50Latency, report.P75Latency, report.P95Latency, report.P99Latency)
					iterations += report.Iterations
					total.Iterations += report.Iterations
					total.Duration += report.Duration
					total.MeanLatency += report.MeanLatency
					total.P50Latency += report.P50Latency
					total.P75Latency += report.P75Latency
					total.P95Latency += report.P95Latency
					total.P99Latency += report.P99Latency
					count++
				}
			}
			fmt.Fprintf(writer, "TOTAL\t%d\t%s\t%f/sec\t%s\t%s\t%s\t%s\t%s\n", total.Iterations, total.Duration,
				float64(total.Iterations)/(float64(total.Duration)/float64(time.Second)),
				total.MeanLatency/time.Duration(count), report.P50Latency/time.Duration(count),
				report.P75Latency/time.Duration(count), report.P95Latency/time.Duration(count),
				report.P99Latency/time.Duration(count))
			writer.Flush()
			uiwriter.Flush()

			if !canceled && maxIterations > 0 && iterations > maxIterations {
				cancel()
				canceled = true
			}
		case <-signalCh:
			if !canceled {
				cancel()
				canceled = true
			}
		}
	}
}

func runBenchmarkWorker(ctx context.Context, job job.Job[benchmark.Config], worker int, ch chan<- workerReport, timeout time.Duration) error {
	job.ID = fmt.Sprintf("%s-worker-%d", job.ID, worker)
	job.Config.Type = benchmark.WorkerType
	job.CreateNamespace = false
	job.DeleteNamespace = false

	step := logging.NewStep(job.ID, "Setting up worker %d", worker)
	step.Start()
	if err := job.Create(ctx, step); err != nil {
		step.Fail(err)
		return err
	}
	step.Complete()

	step = logging.NewStep(job.ID, "Running worker %d", worker)
	step.Start()
	stream, err := job.GetLogs(ctx)
	if err != nil {
		step.Fail(err)
		return err
	}
	defer stream.Close()

	scanner := bufio.NewScanner(stream)
	for scanner.Scan() {
		var report benchmark.Report
		if err := json.Unmarshal(scanner.Bytes(), &report); err == nil {
			ch <- workerReport{
				Report: report,
				worker: worker,
			}
		}
	}
	step.Complete()

	step = logging.NewStep(job.ID, "Tearing down worker %d", worker)
	step.Start()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := job.Echo(ctx, shutdownFile, []byte(job.ID)); err != nil {
		step.Fail(err)
		return err
	}
	if _, _, err := job.GetStatus(ctx); err != nil {
		step.Fail(err)
		return err
	}
	if err := job.Delete(ctx, step); err != nil {
		step.Fail(err)
		return err
	}
	step.Complete()
	return nil
}

func tearDownBenchmark(job job.Job[benchmark.Config], timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	job.Config.Type = benchmark.TearDownType
	job.CreateNamespace = false
	step := logging.NewStep(job.ID, "Tearing down benchmark")
	step.Start()
	if err := runJob(ctx, job, step); err != nil {
		step.Fail(err)
		return err
	}
	step.Complete()
	return nil
}

type workerReport struct {
	benchmark.Report
	worker int
}
