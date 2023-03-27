// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package benchmark

import (
	"context"
	"github.com/onosproject/helmit/pkg/helm"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// BenchmarkingSuite is a suite of benchmarks
type BenchmarkingSuite interface {
	// SetNamespace sets the suite namespace
	SetNamespace(namespace string)
	// Namespace returns the suite namespace
	Namespace() string
	// SetConfig sets the Kubernetes REST configuration
	SetConfig(config *rest.Config)
	// Config returns the Kubernetes REST configuration
	Config() *rest.Config
	// SetHelm sets the Helm client
	SetHelm(helm *helm.Helm)
	// Helm returns the Helm client
	Helm() *helm.Helm
}

// Suite is the base for a benchmark suite
type Suite struct {
	*kubernetes.Clientset
	namespace string
	config    *rest.Config
	helm      *helm.Helm
}

// SetNamespace sets the suite namespace
func (suite *Suite) SetNamespace(namespace string) {
	suite.namespace = namespace
}

// Namespace returns the suite namespace
func (suite *Suite) Namespace() string {
	return suite.namespace
}

// SetConfig sets the Kubernetes REST configuration
func (suite *Suite) SetConfig(config *rest.Config) {
	suite.config = config
	suite.Clientset = kubernetes.NewForConfigOrDie(config)
}

// Config returns the Kubernetes REST configuration
func (suite *Suite) Config() *rest.Config {
	return suite.config
}

// SetHelm sets the Helm client
func (suite *Suite) SetHelm(helm *helm.Helm) {
	suite.helm = helm
}

// Helm returns the Helm client
func (suite *Suite) Helm() *helm.Helm {
	return suite.helm
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

// InternalBenchmarkSuite is an internal named benchmark suite
type InternalBenchmarkSuite struct {
	Name  string
	Suite BenchmarkingSuite
}
