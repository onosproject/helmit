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
	SetNamespace(namespace string)
	Namespace() string
	SetConfig(config *rest.Config)
	Config() *rest.Config
	SetHelm(helm *helm.Helm)
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

// SetupSuite has a SetupSuite method, which will run before the
// tests in the suite are run.
type SetupSuite interface {
	SetupSuite(ctx context.Context) error
}

// SetupTest has a SetupTest method, which will run before each
// test in the suite.
type SetupTest interface {
	SetupTest(ctx context.Context) error
}

// TearDownSuite has a TearDownSuite method, which will run after
// all the tests in the suite have been run.
type TearDownSuite interface {
	TearDownSuite(ctx context.Context) error
}

// TearDownTest has a TearDownTest method, which will run after
// each test in the suite.
type TearDownTest interface {
	TearDownTest(ctx context.Context) error
}
