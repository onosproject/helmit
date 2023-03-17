// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"errors"
	petname "github.com/dustinkirkland/golang-petname"
	"github.com/onosproject/helmit/internal/console"
	"github.com/onosproject/helmit/pkg/bench"
	"os"
	"path/filepath"
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
		Use:     "benchmark",
		Aliases: []string{"benchmarks", "bench"},
		Short:   "Run benchmarks on Kubernetes",
		Example: benchExamples,
		Args:    cobra.MaximumNArgs(1),
		RunE:    runBenchCommand,
	}
	cmd.Flags().StringP("namespace", "n", "default", "the namespace in which to run the benchmarks")
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
	cmd.Flags().DurationP("max-latency", "m", 0, "maximum latency allowed")
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
	verbose, _ := cmd.Flags().GetBool("verbose")

	var opts []console.Option
	if verbose {
		opts = append(opts, console.WithVerbose())
	}

	context := console.NewContext(cmd.OutOrStdout(), opts...)
	defer context.Close()

	pkgPath := ""
	if len(args) > 0 {
		pkgPath = args[0]
	}

	namespace, _ := cmd.Flags().GetString("namespace")
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

	// Either --iterations or --duration must be specified
	if iterations == 0 && duration == 0 {
		return errors.New("either --iterations or --duration must be specified")
	}

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

	var d *time.Duration
	if duration != 0 {
		d = &duration
	}

	secrets, err := parseSecrets(secretsArray)
	if err != nil {
		return err
	}

	// Generate a unique benchmark ID
	benchID := petname.Generate(2, "-")

	var executable string
	if pkgPath != "" {
		executable = filepath.Join(os.TempDir(), "helmit", benchID)
		if image == "" {
			image = "onosproject/helmit-runner:latest"
		}
		err := context.Fork("Preparing artifacts", func(context *console.Context) error {
			err := context.Run(func(status *console.Status) error {
				status.Reportf("Validating package path %s", pkgPath)
				return validatePackage(pkgPath)
			}).Wait()
			if err != nil {
				return err
			}

			err = context.Run(func(status *console.Status) error {
				status.Reportf("Building %s", executable)
				return buildBinary(pkgPath, executable)
			}).Wait()
			if err != nil {
				cmd.SilenceUsage = true
				cmd.SilenceErrors = true
				return err
			}
			return nil
		}).Join()
		if err != nil {
			return err
		}
	}

	manager := job.NewManager[bench.Config](job.ExecutorType)
	job := job.Job[bench.Config]{
		Spec: job.Spec{
			ID:              benchID,
			Namespace:       namespace,
			ServiceAccount:  serviceAccount,
			Labels:          labels,
			Annotations:     annotations,
			Executable:      executable,
			Image:           image,
			ImagePullPolicy: pullPolicy,
			Context:         contextPath,
			ValueFiles:      valueFiles,
			Values:          values,
			Timeout:         timeout,
			NoTeardown:      noTeardown,
			Secrets:         secrets,
		},
		Config: bench.Config{
			WorkerConfig: bench.WorkerConfig{
				Image:           image,
				ImagePullPolicy: pullPolicy,
				// TODO: Add environment variables?
				// Env: ...
			},
			Suite:          suite,
			Benchmark:      benchmarkName,
			Workers:        workers,
			Parallelism:    parallelism,
			Iterations:     iterations,
			Duration:       d,
			ReportInterval: reportInterval,
			Args:           benchArgs,
			NoTeardown:     noTeardown,
		},
	}

	err = context.Fork("Starting benchmark", func(context *console.Context) error {
		return manager.Start(job, context)
	}).Join()
	if err != nil {
		return err
	}

	err = context.Fork("Running benchmark", func(context *console.Context) error {
		return manager.Run(job, context)
	}).Join()
	if err != nil {
		return err
	}

	err = context.Fork("Terminating benchmark", func(context *console.Context) error {
		code, err := manager.Stop(job)
		if err != nil {
			return err
		}
		os.Exit(code)
		return nil
	}).Join()
	if err != nil {
		return err
	}
	return nil
}
