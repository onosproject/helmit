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
	"strconv"
	"strings"
	"time"

	"github.com/onosproject/helmit/pkg/job"
	"github.com/onosproject/helmit/pkg/simulation"

	"github.com/onosproject/helmit/pkg/util/random"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
)

var (
	simulateExample = `
		# Simulate operations on an Atomix map
		helmit simulate --image atomix/kubernetes-simulations --simulation map --duration 1m

		# Configure the simulated Atomix cluster
		helmit simulate --image atomix/kubernetes-simulations --simulation map --duration 1m --set raft.clusters=3 --set raft.partitions=3

		# Configure scheduled operations on an Atomix map
		helmit simulate --image atomix/kubernetes-simulations --simulation map --schedule put=2s --schedule get=1s,.5 --schedule remove=5s --duration 5m

		# Verify an Atomix map simulation against a TLA+ model
		helmit simulate --image atomix/kubernetes-simulations --simulation map --duration 5m --verify --model models/MapCacheTrace.tla --module models/MapHistory.tla --spec Spec --invariant StateInvariant`
)

func getSimulateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "simulate",
		Aliases: []string{"sim", "simulation"},
		Short:   "Run simulations on Kubernetes",
		Example: simulateExample,
		RunE:    runSimulateCommand,
	}
	cmd.Flags().StringP("context", "c", "", "the simulation context")
	cmd.Flags().StringP("image", "i", "", "the simulation image to run")
	cmd.Flags().String("image-pull-policy", string(corev1.PullIfNotPresent), "the Docker image pull policy")
	cmd.Flags().StringArrayP("values", "f", []string{}, "release values paths")
	cmd.Flags().StringArray("set", []string{}, "cluster argument overrides")
	cmd.Flags().StringP("simulation", "s", "", "the simulation to run")
	cmd.Flags().IntP("simulators", "w", 1, "the number of simulator workers to run")
	cmd.Flags().DurationP("duration", "d", 10*time.Minute, "the duration for which to run the simulation")
	cmd.Flags().StringToStringP("args", "a", map[string]string{}, "a mapping of named simulation arguments")
	cmd.Flags().StringToStringP("schedule", "r", map[string]string{}, "a mapping of operations to schedule")
	return cmd
}

func runSimulateCommand(cmd *cobra.Command, args []string) error {
	setupCommand(cmd)

	pkgPath := ""
	if len(args) > 0 {
		pkgPath = args[0]
	}

	context, _ := cmd.Flags().GetString("context")
	image, _ := cmd.Flags().GetString("image")
	sim, _ := cmd.Flags().GetString("simulation")
	workers, _ := cmd.Flags().GetInt("simulators")
	duration, _ := cmd.Flags().GetDuration("duration")
	timeout, _ := cmd.Flags().GetDuration("timeout")
	files, _ := cmd.Flags().GetStringArray("values")
	sets, _ := cmd.Flags().GetStringArray("set")
	simArgs, _ := cmd.Flags().GetStringToString("args")
	operations, _ := cmd.Flags().GetStringToString("schedule")
	imagePullPolicy, _ := cmd.Flags().GetString("image-pull-policy")
	pullPolicy := corev1.PullPolicy(imagePullPolicy)

	// Either a command package or image must be specified
	if pkgPath == "" && image == "" {
		return errors.New("must specify either a simulation package or --image to run")
	}

	// Generate a unique simulation ID
	simID := random.NewPetName(2)

	// If a command package was provided, build a binary and update the image tag
	var executable string
	if pkgPath != "" {
		executable = filepath.Join(os.TempDir(), "helmit", simID)
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

	// Parse the rate and jitter flags
	rates := make(map[string]time.Duration)
	jitters := make(map[string]float64)
	for name, value := range operations {
		var rate string
		index := strings.Index(value, ",")
		if index == -1 {
			rate = value
		} else {
			rate = value[:index]
			jitter := value[index+1:]
			f, err := strconv.ParseFloat(jitter, 64)
			if err != nil {
				return err
			}
			jitters[name] = f
		}
		d, err := time.ParseDuration(rate)
		if err != nil {
			return err
		}
		rates[name] = d
	}

	valueFiles, err := parseFiles(files)
	if err != nil {
		return err
	}

	values, err := parseOverrides(sets)
	if err != nil {
		return err
	}

	config := &simulation.Config{
		Config: &job.Config{
			ID:              simID,
			Image:           image,
			ImagePullPolicy: pullPolicy,
			Executable:      executable,
			Context:         context,
			ValueFiles:      valueFiles,
			Values:          values,
			Timeout:         timeout,
		},
		Simulation: sim,
		Simulators: workers,
		Duration:   duration,
		Rates:      rates,
		Jitter:     jitters,
		Args:       simArgs,
	}
	return simulation.Run(config)
}
