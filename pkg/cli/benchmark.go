// Copyright 2020-present Open Networking Foundation.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cli

import (
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/onosproject/helmit/pkg/job"

	"github.com/onosproject/helmit/pkg/benchmark"
	"github.com/onosproject/helmit/pkg/util/random"
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
	cmd.Flags().StringToStringP("args", "a", map[string]string{}, "a mapping of named benchmark arguments")
	cmd.Flags().Duration("timeout", 10*time.Minute, "benchmark timeout")
	cmd.Flags().Bool("no-teardown", false, "do not tear down clusters following benchmarks")
	return cmd
}

func runBenchCommand(cmd *cobra.Command, args []string) error {
	setupCommand(cmd)

	pkgPath := ""
	if len(args) > 0 {
		pkgPath = args[0]
	}

	namespace, _ := cmd.Flags().GetString("namespace")
	serviceAccount, _ := cmd.Flags().GetString("service-account")
	context, _ := cmd.Flags().GetString("context")
	image, _ := cmd.Flags().GetString("image")
	suite, _ := cmd.Flags().GetString("suite")
	benchmarkName, _ := cmd.Flags().GetString("benchmark")
	workers, _ := cmd.Flags().GetInt("workers")
	parallelism, _ := cmd.Flags().GetInt("parallel")
	iterations, _ := cmd.Flags().GetInt("iterations")
	duration, _ := cmd.Flags().GetDuration("duration")
	files, _ := cmd.Flags().GetStringArray("values")
	sets, _ := cmd.Flags().GetStringArray("set")
	benchArgs, _ := cmd.Flags().GetStringToString("args")
	timeout, _ := cmd.Flags().GetDuration("timeout")
	imagePullPolicy, _ := cmd.Flags().GetString("image-pull-policy")
	pullPolicy := corev1.PullPolicy(imagePullPolicy)
	noTeardown, _ := cmd.Flags().GetBool("no-teardown")

	// Either --iterations or --duration must be specified
	if iterations == 0 && duration == 0 {
		return errors.New("either --iterations or --duration must be specified")
	}

	// Either a command package or image must be specified
	if pkgPath == "" && image == "" {
		return errors.New("must specify either a benchmark package or --image to run")
	}

	// Generate a unique benchmark ID
	benchID := random.NewPetName(2)

	// If a command package was provided, build a binary and update the image tag
	var executable string
	if pkgPath != "" {
		executable = filepath.Join(os.TempDir(), "helmit", benchID)
		err := buildBinary(pkgPath, executable)
		if err != nil {
			cmd.SilenceUsage = true
			cmd.SilenceErrors = true
			return err
		}
		if image == "" {
			image = "onosproject/helmit-runner:latest"
		}
	}

	// If a context was provided, convert the context to its absolute path
	if context != "" {
		path, err := filepath.Abs(context)
		if err != nil {
			return err
		}
		context = path
	}

	var maxLatency *time.Duration
	if cmd.Flags().Changed("max-latency") {
		d, _ := cmd.Flags().GetDuration("max-latency")
		maxLatency = &d
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

	config := &benchmark.Config{
		Config: &job.Config{
			ID:              benchID,
			Namespace:       namespace,
			ServiceAccount:  serviceAccount,
			Executable:      executable,
			Image:           image,
			ImagePullPolicy: pullPolicy,
			Context:         context,
			ValueFiles:      valueFiles,
			Values:          values,
			Timeout:         timeout,
			NoTeardown:      noTeardown,
		},
		Suite:       suite,
		Benchmark:   benchmarkName,
		Workers:     workers,
		Parallelism: parallelism,
		Iterations:  iterations,
		Duration:    d,
		Args:        benchArgs,
		MaxLatency:  maxLatency,
		NoTeardown: noTeardown,
	}
	return benchmark.Run(config)
}
