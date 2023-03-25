// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package bench

import (
	"context"
	"github.com/onosproject/helmit/pkg/helm"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// BenchmarkingSuite is a suite of benchmarks
type BenchmarkingSuite interface {
	SetNamespace(namespace string)
	Namespace() string
	SetConfig(config *rest.Config)
	Config() *rest.Config
	SetHelm(helm *helm.Helm)
	Helm() *helm.Helm
}

// Suite is the base for a benchmark suite
type Suite struct {
	*kubernetes.Clientset
	namespace string
	config    *rest.Config
	helm      *helm.Helm
}

func (suite *Suite) SetNamespace(namespace string) {
	suite.namespace = namespace
}

func (suite *Suite) Namespace() string {
	return suite.namespace
}

func (suite *Suite) SetConfig(config *rest.Config) {
	suite.config = config
	suite.Clientset = kubernetes.NewForConfigOrDie(config)
}

func (suite *Suite) Config() *rest.Config {
	return suite.config
}

func (suite *Suite) SetHelm(helm *helm.Helm) {
	suite.helm = helm
}

func (suite *Suite) Helm() *helm.Helm {
	return suite.helm
}

// SetupSuite is an interface for setting up a suite of benchmarks
type SetupSuite interface {
	SetupSuite(ctx context.Context) error
}

// TearDownSuite is an interface for tearing down a suite of benchmarks
type TearDownSuite interface {
	TearDownSuite(ctx context.Context) error
}

// SetupWorker is an interface for setting up individual benchmarks
type SetupWorker interface {
	SetupWorker(ctx context.Context) error
}

// TearDownWorker is an interface for tearing down individual benchmarks
type TearDownWorker interface {
	TearDownWorker(ctx context.Context) error
}

// SetupBenchmark is an interface for executing code before every benchmark
type SetupBenchmark interface {
	SetupBenchmark(ctx context.Context) error
}

// TearDownBenchmark is an interface for executing code after every benchmark
type TearDownBenchmark interface {
	TearDownBenchmark(ctx context.Context) error
}
