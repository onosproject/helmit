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

// Suite is the base for a test suite
type Suite struct {
	suite.Suite
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

// SetupSuite has a SetupSuite method, which will run before the
// tests in the suite are run.
type SetupSuite interface {
	// SetupSuite is called at the beginning of a test run to set up the test suite
	SetupSuite(ctx context.Context)
}

// TearDownSuite has a TearDownSuite method, which will run after
// all the tests in the suite have been run.
type TearDownSuite interface {
	// TearDownSuite is called at the end of a test run to tear down the test suite
	TearDownSuite(ctx context.Context)
}

// SetupTest has a SetupTest method, which will run before each
// test in the suite.
type SetupTest interface {
	// SetupTest is called at the beginning of each test run to set up the test
	SetupTest(ctx context.Context)
}

// TearDownTest has a TearDownTest method, which will run after
// each test in the suite.
type TearDownTest interface {
	// TearDownTest is called at the end of each test run to tear down the test
	TearDownTest(ctx context.Context)
}

// InternalTestSuite is an internal named test suite
type InternalTestSuite struct {
	Name  string
	Suite TestingSuite
}
