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
	"github.com/onosproject/helmit/internal/logging"
	"github.com/onosproject/helmit/pkg/bench"
	"os"
	"path/filepath"
	"sync"
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

func getBenchCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "bench",
		Aliases: []string{"benchmark", "benchmarks"},
		Short:   "Run benchmarks on Kubernetes",
		Example: benchExamples,
		Args:    cobra.MaximumNArgs(1),
		RunE:    runBenchCommand,
	}
	cmd.Flags().StringP("namespace", "n", "default", "the namespace in which to run the benchmarks")
	cmd.Flags().Bool("create-namespace", false, "whether to create the namespace when running the test")
	cmd.Flags().String("service-account", "", "the name of the service account to use to run worker pods")
	cmd.Flags().StringToString("labels", map[string]string{}, "a mapping of labels to add to the test pod")
	cmd.Flags().StringToString("annotations", map[string]string{}, "a mapping of annotations to add to the test pod")
	cmd.Flags().StringP("context", "c", "", "the benchmark context")
	cmd.Flags().StringP("image", "i", "", "the benchmark image to run")
	cmd.Flags().String("image-pull-policy", string(corev1.PullIfNotPresent), "the Docker image pull policy")
	cmd.Flags().StringArrayP("values", "f", []string{}, "release values paths")
	cmd.Flags().StringArray("set", []string{}, "cluster argument overrides")
	cmd.Flags().StringP("suite", "s", "", "the benchmark suite to run")
	cmd.Flags().StringP("benchmark", "b", "", "the name of the benchmark to run")
	cmd.Flags().IntP("workers", "w", 1, "the number of workers to run")
	cmd.Flags().Int("parallel", 1, "the number of concurrent goroutines per client")
	cmd.Flags().IntP("iterations", "", 0, "the number of iterations to run")
	cmd.Flags().DurationP("duration", "d", 0, "the duration for which to run the test")
	cmd.Flags().DurationP("report-interval", "r", 5*time.Second, "the interval at which to report benchmark results")
	cmd.Flags().StringToStringP("args", "a", map[string]string{}, "a mapping of named benchmark arguments")
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

	pkgPath := ""
	if len(args) > 0 {
		pkgPath = args[0]
	}

	namespace, _ := cmd.Flags().GetString("namespace")
	createNamespace, _ := cmd.Flags().GetBool("create-namespace")
	serviceAccount, _ := cmd.Flags().GetString("service-account")
	labels, _ := cmd.Flags().GetStringToString("labels")
	annotations, _ := cmd.Flags().GetStringToString("annotations")
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
	if pkgPath == "" && image == "" {
		return errors.New("must specify either a benchmark package or --image to run")
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

	// Generate a unique benchmark ID
	benchID := petname.Generate(2, "-")

	var executable string
	if pkgPath != "" {
		step := logging.NewStep(benchID, "Preparing artifacts")
		step.Start()
		executable = filepath.Join(os.TempDir(), "helmit", benchID)
		defer os.RemoveAll(executable)
		image = defaultRunnerImage

		step.Logf("Validating %s", pkgPath)
		if err := validatePackage(pkgPath); err != nil {
			step.Fail(err)
			return err
		}
		step.Logf("Building %s", pkgPath)
		if err := buildBinary(pkgPath, executable); err != nil {
			step.Fail(err)
			return err
		}
		step.Complete()
	}

	config := bench.Config{
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
		config.Context = defaultContextPath
	}

	if len(valueFiles) > 0 {
		config.ValueFiles = make(map[string][]string)
		for release, files := range valueFiles {
			var baseFiles []string
			for _, file := range files {
				baseFiles = append(baseFiles, filepath.Base(file))
			}
			config.ValueFiles[release] = baseFiles
		}
	}

	job := job.Job[bench.Config]{
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

	ctx, cancel := context.WithCancel(context.Background())
	if duration > 0 {
		ctx, cancel = context.WithTimeout(ctx, duration)
	}
	defer cancel()

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

func runJob(ctx context.Context, job job.Job[bench.Config], log logging.Logger) error {
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

func setupBenchmark(job job.Job[bench.Config], timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	job.Config.Type = bench.SetupType
	step := logging.NewStep(job.ID, "Setting up benchmark")
	step.Start()
	if err := runJob(ctx, job, step); err != nil {
		step.Fail(err)
		return err
	}
	step.Complete()
	return nil
}

func runBenchmark(job job.Job[bench.Config], workers int, iterations int, duration time.Duration, timeout time.Duration) error {
	ctx, cancel := context.WithCancel(context.Background())
	if duration > 0 {
		ctx, cancel = context.WithTimeout(ctx, duration)
	}
	defer cancel()

	ch := make(chan workerReport)
	wg := &sync.WaitGroup{}
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(worker int) {
			_ = runBenchmarkWorker(ctx, job, worker, ch, timeout)
			wg.Done()
		}(i)
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	uiwriter := uilive.New()
	uiwriter.Out = os.Stdout

	reports := make([]*workerReport, workers)
	var canceled bool
	for report := range ch {
		reports[report.worker] = &report

		writer := new(tabwriter.Writer)
		writer.Init(uiwriter, 0, 0, 3, ' ', tabwriter.FilterHTML)

		fmt.Fprintln(writer, "WORKER\tITERATIONS\tDURATION\tTHROUGHPUT\tMEAN LATENCY\tMEDIAN LATENCY\t75% LATENCY\t95% LATENCY\t99% LATENCY")
		var count int
		var total bench.Report
		for worker, report := range reports {
			if report != nil {
				fmt.Fprintf(writer, "%d\t%d\t%s\t%f/sec\t%s\t%s\t%s\t%s\t%s\n",
					worker, report.Iterations, report.Duration,
					float64(report.Iterations)/(float64(report.Duration)/float64(time.Second)),
					report.MeanLatency, report.P50Latency, report.P75Latency, report.P95Latency, report.P99Latency)
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

		if !canceled && iterations > 0 && total.Iterations > iterations {
			cancel()
			canceled = true
		}
	}
	return nil
}

func runBenchmarkWorker(ctx context.Context, job job.Job[bench.Config], worker int, ch chan<- workerReport, timeout time.Duration) error {
	job.ID = fmt.Sprintf("%s-worker-%d", job.ID, worker)
	job.Config.Type = bench.WorkerType
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
		var report bench.Report
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
	if err := job.Delete(ctx, step); err != nil {
		step.Fail(err)
		return err
	}
	step.Complete()
	return nil
}

func tearDownBenchmark(job job.Job[bench.Config], timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	job.Config.Type = bench.TearDownType
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
	bench.Report
	worker int
}
