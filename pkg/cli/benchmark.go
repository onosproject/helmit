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
	"github.com/onosproject/helmet/pkg/job"
	"os"
	"path/filepath"
	"time"

	"github.com/onosproject/helmet/pkg/benchmark"
	"github.com/onosproject/helmet/pkg/util/random"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
)

func getBenchCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "benchmark",
		Aliases: []string{"benchmarks", "bench"},
		Short:   "Run benchmarks on Kubernetes",
		Args:    cobra.MaximumNArgs(1),
		RunE:    runBenchCommand,
	}
	cmd.Flags().StringP("package", "p", "", "the package to run")
	cmd.Flags().StringP("image", "i", "", "the benchmark image to run")
	cmd.Flags().String("image-pull-policy", string(corev1.PullIfNotPresent), "the Docker image pull policy")
	cmd.Flags().StringArrayP("values", "f", []string{}, "release values paths")
	cmd.Flags().StringArray("set", []string{}, "cluster argument overrides")
	cmd.Flags().StringP("suite", "s", "", "the benchmark suite to run")
	cmd.Flags().StringP("benchmark", "b", "", "the name of the benchmark to run")
	cmd.Flags().IntP("workers", "w", 1, "the number of workers to run")
	cmd.Flags().Int("parallel", 1, "the number of concurrent goroutines per client")
	cmd.Flags().IntP("requests", "n", 0, "the number of requests to run")
	cmd.Flags().DurationP("max-latency", "m", 0, "maximum latency allowed")
	cmd.Flags().DurationP("duration", "d", 0, "the duration for which to run the test")
	cmd.Flags().StringToStringP("args", "a", map[string]string{}, "a mapping of named benchmark arguments")
	cmd.Flags().Duration("timeout", 10*time.Minute, "benchmark timeout")
	cmd.Flags().Bool("no-teardown", false, "do not tear down clusters following tests")
	return cmd
}

func runBenchCommand(cmd *cobra.Command, args []string) error {
	setupCommand(cmd)

	pkgPath, _ := cmd.Flags().GetString("package")
	image, _ := cmd.Flags().GetString("image")
	suite, _ := cmd.Flags().GetString("suite")
	benchmarkName, _ := cmd.Flags().GetString("benchmark")
	workers, _ := cmd.Flags().GetInt("workers")
	parallelism, _ := cmd.Flags().GetInt("parallel")
	requests, _ := cmd.Flags().GetInt("requests")
	files, _ := cmd.Flags().GetStringArray("values")
	sets, _ := cmd.Flags().GetStringArray("set")
	benchArgs, _ := cmd.Flags().GetStringToString("args")
	timeout, _ := cmd.Flags().GetDuration("timeout")
	imagePullPolicy, _ := cmd.Flags().GetString("image-pull-policy")
	pullPolicy := corev1.PullPolicy(imagePullPolicy)

	if pkgPath == "" && image == "" {
		return errors.New("must specify either a --package or --image to run")
	}

	var duration *time.Duration
	if cmd.Flags().Changed("duration") {
		d, _ := cmd.Flags().GetDuration("duration")
		duration = &d
		if timeout <= d {
			timeout = d * 2
		}
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

	testID := random.NewPetName(2)

	var executable string
	if pkgPath != "" {
		executable = filepath.Join(os.TempDir(), "helmet", testID)
		err = buildBinary(pkgPath, executable)
		if err != nil {
			cmd.SilenceUsage = true
			cmd.SilenceErrors = true
			return err
		}
		if image == "" {
			image = "onosproject/helmet-runner:latest"
		}
	}

	var context string
	if len(args) > 0 {
		path, err := filepath.Abs(args[0])
		if err != nil {
			return err
		}
		context = path
	}

	config := &benchmark.Config{
		Config: &job.Config{
			ID:              testID,
			Executable:      executable,
			Image:           image,
			ImagePullPolicy: corev1.PullPolicy(pullPolicy),
			Context:         context,
			ValueFiles:      valueFiles,
			Values:          values,
			Timeout:         timeout,
		},
		Suite:       suite,
		Benchmark:   benchmarkName,
		Workers:     workers,
		Parallelism: parallelism,
		Requests:    requests,
		Duration:    duration,
		Args:        benchArgs,
		MaxLatency:  maxLatency,
	}
	return benchmark.Run(config)
}
