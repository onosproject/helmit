// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package test

import (
	"context"
	"github.com/onosproject/helmit/pkg/helm"
	"github.com/stretchr/testify/suite"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// TestingSuite is a suite of tests
type TestingSuite interface {
	suite.TestingSuite
	SetConfig(config *rest.Config)
	Config() *rest.Config
	SetHelm(helm *helm.Helm)
	Helm() *helm.Helm
}

// Suite is the base for a test suite
type Suite struct {
	suite.Suite
	*kubernetes.Clientset
	config *rest.Config
	helm   *helm.Helm
}

func (suite *Suite) Namespace() string {
	return suite.helm.Namespace()
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

// SetupSuite is an interface for setting up a suite of tests
type SetupSuite interface {
	SetupSuite(ctx context.Context) error
}

// SetupTest is an interface for setting up individual tests
type SetupTest interface {
	SetupTest(ctx context.Context) error
}

// TearDownSuite is an interface for tearing down a suite of tests
type TearDownSuite interface {
	TearDownSuite(ctx context.Context) error
}

// TearDownTest is an interface for tearing down individual tests
type TearDownTest interface {
	TearDownTest(ctx context.Context) error
}

// BeforeTest is an interface for executing code before every test
type BeforeTest interface {
	BeforeTest(testName string) error
}

// AfterTest is an interface for executing code after every test
type AfterTest interface {
	AfterTest(testName string) error
}
