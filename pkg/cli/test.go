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
	"go/build"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/onosproject/helmit/pkg/job"

	"github.com/onosproject/helmit/pkg/util/logging"

	"github.com/onosproject/helmit/pkg/test"
	"github.com/onosproject/helmit/pkg/util/random"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

const testExamples = `
  # Run tests packaged in a Docker image.
  helmit test --image atomix/kubernetes-tests:latest

  # Run tests by referencing a command package and providing a context.
  # The specified context will be loaded into the test pod as the current working directory.
  helmit test ./cmd/tests --context ./charts

  # Run tests in a specific namespace.
  helmit test ./cmd/tests -n integration-tests

  # Run a test suite by name.
  helmit test ./cmd/tests -c ./charts --suite atomix

  # Run a single test by name.
  helmit test ./cmd/tests -c ./charts --suite atomix --test TestMap

  # Override Helm chart values with flags.
  # Value overrids must be namespaced with the name of the release to which to apply the value.
  helmit test ./cmd/tests -c ./charts --set atomix-controller.image=atomix/atomix-controller:latest --set atomix-raft.replicas=3 --suite atomix

  # Override Helm chart values with values files.
  # Values files must be key/value pairs where the key is the Helm release name and the value the path to the file.
  helmit test ./cmd/tests -c ./charts -f atomix-controller=./atomix-controller.yaml --suite atomix
`

func getTestCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "test",
		Aliases: []string{"tests"},
		Short:   "Run tests on Kubernetes",
		Example: testExamples,
		Args:    cobra.MaximumNArgs(1),
		RunE:    runTestCommand,
	}
	cmd.Flags().StringP("namespace", "n", "default", "the namespace in which to run the tests")
	cmd.Flags().String("service-account", "", "the name of the service account to use to run test pods")
	cmd.Flags().StringP("context", "c", "", "the test context")
	cmd.Flags().StringP("image", "i", "", "the test image to run")
	cmd.Flags().String("image-pull-policy", string(corev1.PullIfNotPresent), "the Docker image pull policy")
	cmd.Flags().StringArrayP("values", "f", []string{}, "release values paths")
	cmd.Flags().StringArray("set", []string{}, "chart value overrides")
	cmd.Flags().StringSliceP("suite", "s", []string{}, "the name of test suite to run")
	cmd.Flags().StringSliceP("test", "t", []string{}, "the name of the test method to run")
	cmd.Flags().Duration("timeout", 10*time.Minute, "test timeout")
	cmd.Flags().Int("iterations", 1, "number of iterations")
	cmd.Flags().Bool("until-failure", false, "run until an error is detected")
	cmd.Flags().Bool("no-teardown", false, "do not tear down clusters following tests")
	cmd.Flags().StringSlice("secret", []string{}, "secrets to pass to the kubernetes pod")
	return cmd
}

func runTestCommand(cmd *cobra.Command, args []string) error {
	setupCommand(cmd)
	pkgPath := ""
	if len(args) > 0 {
		pkgPath = args[0]
	}

	namespace, _ := cmd.Flags().GetString("namespace")
	serviceAccount, _ := cmd.Flags().GetString("service-account")
	context, _ := cmd.Flags().GetString("context")
	image, _ := cmd.Flags().GetString("image")
	files, _ := cmd.Flags().GetStringArray("values")
	sets, _ := cmd.Flags().GetStringArray("set")
	suites, _ := cmd.Flags().GetStringSlice("suite")
	testNames, _ := cmd.Flags().GetStringSlice("test")
	timeout, _ := cmd.Flags().GetDuration("timeout")
	pullPolicy, _ := cmd.Flags().GetString("image-pull-policy")
	iterations, _ := cmd.Flags().GetInt("iterations")
	untilFailure, _ := cmd.Flags().GetBool("until-failure")
	noTeardown, _ := cmd.Flags().GetBool("no-teardown")
	secretsArray, _ := cmd.Flags().GetStringSlice("secret")

	// Either a command package or image must be specified
	if pkgPath == "" && image == "" {
		return errors.New("must specify either a test package or --image to run")
	}

	// Generate a unique test ID
	testID := random.NewPetName(2)

	// If a command package was provided, build a binary and update the image tag
	var executable string
	if pkgPath != "" {
		executable = filepath.Join(os.TempDir(), "helmit", testID)
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

	if untilFailure {
		iterations = -1
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

	config := &test.Config{
		Config: &job.Config{
			ID:              testID,
			ServiceAccount:  serviceAccount,
			Namespace:       namespace,
			Image:           image,
			ImagePullPolicy: corev1.PullPolicy(pullPolicy),
			Executable:      executable,
			Context:         context,
			ValueFiles:      valueFiles,
			Values:          values,
			Timeout:         timeout,
			NoTeardown:      noTeardown,
			Secrets:         secrets,
		},
		Suites:     suites,
		Tests:      testNames,
		Iterations: iterations,
		Verbose:    logging.GetVerbose(),
		NoTeardown: noTeardown,
	}
	return test.Run(config)
}

func buildBinary(pkgPath, binPath string) error {
	workDir, err := os.Getwd()
	if err != nil {
		return err
	}

	pkg, err := build.Import(pkgPath, workDir, build.ImportComment)
	if err != nil {
		return err
	}

	if !pkg.IsCommand() {
		return errors.New("test package must be a command")
	}

	// Build the command
	build := exec.Command("go", "build", "-o", binPath, pkgPath)
	build.Stderr = os.Stderr
	build.Stdout = os.Stdout
	env := os.Environ()
	env = append(env, "GOOS=linux", "CGO_ENABLED=0")
	build.Env = env
	return build.Run()
}

func parseFiles(files []string) (map[string][]string, error) {
	if len(files) == 0 {
		return map[string][]string{}, nil
	}

	values := make(map[string][]string)
	for _, path := range files {
		index := strings.Index(path, "=")
		if index == -1 {
			return nil, errors.New("values file must be in the format {release}={file}")
		}
		release, path := path[:index], path[index+1:]
		path, err := filepath.Abs(path)
		if err != nil {
			return nil, err
		}
		_, err = os.Stat(path)
		if err != nil {
			return nil, err
		}
		releaseValues, ok := values[release]
		if !ok {
			releaseValues = make([]string, 0)
		}
		values[release] = append(releaseValues, path)
	}
	return values, nil
}

func parseOverrides(values []string) (map[string][]string, error) {
	overrides := make(map[string][]string)
	for _, set := range values {
		index := strings.Index(set, ".")
		if index == -1 {
			return nil, errors.New("values must be in the format {release}.{path}={value}")
		}
		release, value := set[:index], set[index+1:]
		override, ok := overrides[release]
		if !ok {
			override = make([]string, 0)
		}
		overrides[release] = append(override, value)
	}
	return overrides, nil
}

func parseSecrets(secrets []string) (map[string]string, error) {
	if len(secrets) == 0 {
		return map[string]string{}, nil
	}

	values := make(map[string]string)
	for _, secret := range secrets {
		index := strings.Index(secret, "=")
		if index == -1 {
			return nil, errors.New("secrets must be in the format {key}={value}")
		}
		key, value := secret[:index], secret[index+1:]
		values[key] = value
	}
	return values, nil
}