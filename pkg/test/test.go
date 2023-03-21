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
	SetContext(ctx context.Context)
	Context() context.Context
}

// Suite is the base for a test suite
type Suite struct {
	suite.Suite
	*kubernetes.Clientset
	ctx       context.Context
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

func (suite *Suite) SetContext(ctx context.Context) {
	suite.ctx = ctx
}

func (suite *Suite) Context() context.Context {
	return suite.ctx
}

func (suite *Suite) SetHelm(helm *helm.Helm) {
	suite.helm = helm
}

func (suite *Suite) Helm() *helm.Helm {
	return suite.helm
}
