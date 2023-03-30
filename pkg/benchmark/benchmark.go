// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package benchmark

import (
	"context"
	"github.com/onosproject/helmit/internal/k8s"
	"github.com/onosproject/helmit/pkg/helm"
	"github.com/onosproject/helmit/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// BenchmarkingSuite is a suite of benchmarks
type BenchmarkingSuite interface {
	// Init initializes the suite
	Init(config Config, secrets map[string]string) error
	// Namespace returns the suite namespace
	Namespace() string
	// Config returns the Kubernetes REST configuration
	Config() *rest.Config
	// Secret returns a secret by name
	Secret(name string) string
	// Secrets returns the injected secrets
	Secrets() map[string]string
	// Arg gets an argument by name
	Arg(name string) types.Value
	// Args returns a map of all test arguments
	Args() map[string]types.Value
	// Helm returns the Helm client
	Helm() *helm.Helm
}

// SetupSuite is an interface for setting up a suite of benchmarks
type SetupSuite interface {
	// SetupSuite is called at the beginning of a benchmark run to set up the benchmark suite
	SetupSuite(ctx context.Context) error
}

// TearDownSuite is an interface for tearing down a suite of benchmarks
type TearDownSuite interface {
	// TearDownSuite is called at the end of a benchmark run to tear down the benchmark suite
	TearDownSuite(ctx context.Context) error
}

// SetupWorker is an interface for setting up individual benchmarks
type SetupWorker interface {
	// SetupWorker is called on each benchmark worker at the start of a benchmark run
	SetupWorker(ctx context.Context) error
}

// TearDownWorker is an interface for tearing down individual benchmarks
type TearDownWorker interface {
	// TearDownWorker is called on each benchmark worker at the end of a benchmark run
	TearDownWorker(ctx context.Context) error
}

// SetupBenchmark is an interface for executing code before every benchmark
type SetupBenchmark interface {
	// SetupBenchmark is called at the beginning of a benchmark run to set up the benchmark
	SetupBenchmark(ctx context.Context) error
}

// TearDownBenchmark is an interface for executing code after every benchmark
type TearDownBenchmark interface {
	// TearDownBenchmark is called at the end of a benchmark run to tear down the benchmark
	TearDownBenchmark(ctx context.Context) error
}

// Suite is the base for a benchmark suite
type Suite struct {
	*kubernetes.Clientset
	config     Config
	secrets    map[string]string
	restConfig *rest.Config
	helm       *helm.Helm
	args       map[string]types.Value
}

// Init initializes the benchmark suite
func (suite *Suite) Init(config Config, secrets map[string]string) error {
	suite.config = config
	suite.secrets = secrets

	args := make(map[string]types.Value)
	for key, value := range config.Args {
		args[key] = types.NewValue(value)
	}
	suite.args = args

	restConfig, err := k8s.GetConfig()
	if err != nil {
		return err
	}
	suite.restConfig = restConfig

	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return err
	}
	suite.Clientset = clientset

	suite.helm = helm.NewClient(helm.Context{
		Namespace:  config.Namespace,
		WorkDir:    config.Context,
		Values:     config.Values,
		ValueFiles: config.ValueFiles,
	})
	return nil
}

// Namespace returns the suite namespace
func (suite *Suite) Namespace() string {
	return suite.config.Namespace
}

// Config returns the Kubernetes REST configuration
func (suite *Suite) Config() *rest.Config {
	return suite.restConfig
}

// Helm returns the Helm client
func (suite *Suite) Helm() *helm.Helm {
	return suite.helm
}

// Secret returns a test secret by name
func (suite *Suite) Secret(name string) string {
	return suite.secrets[name]
}

// Secrets returns the injected secrets
func (suite *Suite) Secrets() map[string]string {
	return suite.secrets
}

// Arg returns a test argument by name
func (suite *Suite) Arg(name string) types.Value {
	value, ok := suite.args[name]
	if !ok {
		return types.NewValue(nil)
	}
	return value
}

// Args returns the test arguments
func (suite *Suite) Args() map[string]types.Value {
	return suite.args
}

var _ BenchmarkingSuite = (*Suite)(nil)
