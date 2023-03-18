// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"errors"
	petname "github.com/dustinkirkland/golang-petname"
	"github.com/onosproject/helmit/internal/console"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"github.com/onosproject/helmit/internal/job"

	"github.com/onosproject/helmit/pkg/test"
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
	cmd.Flags().Bool("create-namespace", false, "whether to create the namespace when running the test")
	cmd.Flags().String("service-account", "", "the name of the service account to use to run test pods")
	cmd.Flags().StringP("context", "c", "", "the test context")
	cmd.Flags().StringP("image", "i", "", "the test image to run")
	cmd.Flags().String("image-pull-policy", string(corev1.PullIfNotPresent), "the Docker image pull policy")
	cmd.Flags().StringArrayP("values", "f", []string{}, "release values paths")
	cmd.Flags().StringArray("set", []string{}, "chart value overrides")
	cmd.Flags().StringSliceP("suite", "s", []string{}, "the name of test suite to run")
	cmd.Flags().StringSliceP("test", "t", []string{}, "the name of the test method to run")
	cmd.Flags().IntP("workers", "w", 1, "the number of workers to run")
	cmd.Flags().BoolP("parallel", "p", false, "whether to run test tests in parallel")
	cmd.Flags().Duration("timeout", 10*time.Minute, "test timeout")
	cmd.Flags().Int("iterations", 1, "number of iterations")
	cmd.Flags().Bool("until-failure", false, "run until an error is detected")
	cmd.Flags().Bool("no-teardown", false, "do not tear down clusters following tests")
	cmd.Flags().StringSlice("secret", []string{}, "secrets to pass to the kubernetes pod")
	cmd.Flags().StringToStringP("args", "a", map[string]string{}, "a mapping of named test arguments")
	return cmd
}

func runTestCommand(cmd *cobra.Command, args []string) error {
	verbose, _ := cmd.Flags().GetBool("verbose")

	var opts []console.Option
	opts = append(opts, console.WithFormat(console.LiveFormat))
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
	createNamespace, _ := cmd.Flags().GetBool("create-namespace")
	serviceAccount, _ := cmd.Flags().GetString("service-account")
	contextPath, _ := cmd.Flags().GetString("context")
	image, _ := cmd.Flags().GetString("image")
	files, _ := cmd.Flags().GetStringArray("values")
	sets, _ := cmd.Flags().GetStringArray("set")
	suites, _ := cmd.Flags().GetStringSlice("suite")
	testNames, _ := cmd.Flags().GetStringSlice("test")
	workers, _ := cmd.Flags().GetInt("workers")
	parallel, _ := cmd.Flags().GetBool("parallel")
	timeout, _ := cmd.Flags().GetDuration("timeout")
	imagePullPolicy, _ := cmd.Flags().GetString("image-pull-policy")
	pullPolicy := corev1.PullPolicy(imagePullPolicy)
	iterations, _ := cmd.Flags().GetInt("iterations")
	untilFailure, _ := cmd.Flags().GetBool("until-failure")
	noTeardown, _ := cmd.Flags().GetBool("no-teardown")
	secretsArray, _ := cmd.Flags().GetStringSlice("secret")
	testArgs, _ := cmd.Flags().GetStringToString("args")

	if untilFailure {
		iterations = -1
	}

	// Either a command package or image must be specified
	if pkgPath == "" && image == "" {
		return errors.New("must specify either a test package or --image to run")
	}
	if image == "" {
		image = defaultRunnerImage
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

	// Generate a unique test ID
	benchID := petname.Generate(2, "-")

	var executable string
	if pkgPath != "" {
		executable = filepath.Join(os.TempDir(), "helmit", benchID)
		defer os.RemoveAll(executable)
		err := context.Fork("Preparing artifacts", func(context *console.Context) error {
			err := context.Run(func(status *console.Status) error {
				status.Reportf("Validating package path %s", pkgPath)
				return validatePackage(pkgPath)
			}).Await()
			if err != nil {
				return err
			}

			err = context.Run(func(status *console.Status) error {
				status.Reportf("Building %s", executable)
				return buildBinary(pkgPath, executable)
			}).Await()
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

	manager := job.NewManager[test.Config](job.ExecutorType)
	job := job.Job[test.Config]{
		Spec: job.Spec{
			ID:              benchID,
			Namespace:       namespace,
			CreateNamespace: createNamespace,
			ServiceAccount:  serviceAccount,
			Executable:      executable,
			Image:           defaultRunnerImage,
			ImagePullPolicy: pullPolicy,
			Context:         contextPath,
			ValueFiles:      valueFiles,
			Values:          values,
			Timeout:         timeout,
			NoTeardown:      noTeardown,
			Secrets:         secrets,
		},
		Config: test.Config{
			WorkerConfig: test.WorkerConfig{
				Image:           image,
				ImagePullPolicy: pullPolicy,
				// TODO: Add environment variables?
				// Env: ...
			},
			Suites:     suites,
			Tests:      testNames,
			Workers:    workers,
			Parallel:   parallel,
			Iterations: iterations,
			Verbose:    verbose,
			Args:       testArgs,
			NoTeardown: noTeardown,
		},
	}

	if err := manager.Start(job, context); err != nil {
		return err
	}
	code, _ := manager.Run(job, context)
	_ = manager.Stop(job, context)
	os.Exit(code)
	return nil
}
