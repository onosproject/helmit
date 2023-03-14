// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package test

import (
	"context"
	"github.com/onosproject/helmit/pkg/helm"
	"github.com/stretchr/testify/suite"
)

// TestingSuite is a suite of tests
type TestingSuite interface {
	suite.TestingSuite
	SetHelm(helm *helm.Helm)
	Helm() *helm.Helm
	SetContext(ctx context.Context)
	Context() context.Context
}

// Suite is the base for a test suite
type Suite struct {
	suite.Suite
	helm *helm.Helm
	ctx  context.Context
}

func (suite *Suite) Namespace() string {
	return suite.helm.Namespace()
}

func (suite *Suite) SetHelm(helm *helm.Helm) {
	suite.helm = helm
}

func (suite *Suite) Helm() *helm.Helm {
	return suite.helm
}

func (suite *Suite) SetContext(ctx context.Context) {
	suite.ctx = ctx
}

func (suite *Suite) Context() context.Context {
	return suite.ctx
}

// SetupTestSuite is an interface for setting up a suite of tests
type SetupTestSuite interface {
	SetupTestSuite() error
}

// SetupTest is an interface for setting up individual tests
type SetupTest interface {
	SetupTest() error
}

// TearDownTestSuite is an interface for tearing down a suite of tests
type TearDownTestSuite interface {
	TearDownTestSuite() error
}

// TearDownTest is an interface for tearing down individual tests
type TearDownTest interface {
	TearDownTest() error
}

// BeforeTest is an interface for executing code before every test
type BeforeTest interface {
	BeforeTest(testName string) error
}

// AfterTest is an interface for executing code after every test
type AfterTest interface {
	AfterTest(testName string) error
}
