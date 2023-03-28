// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package test

import (
	"context"
	"github.com/onosproject/helmit/pkg/test"
	"time"
)

// ChartTestSuite is an example of a Helm chart test suite
type ChartTestSuite struct {
	test.Suite
}

// TestLocalInstall tests a local chart installation
func (s *ChartTestSuite) TestLocalInstall(ctx context.Context) {
	err := s.Helm().Install("atomix-controller", "./controller/chart").
		Set("image.tag", "latest").
		Set("init.image.tag", "latest").
		Wait().
		Do(ctx)
	s.NoError(err)

	err = s.Helm().Uninstall("atomix-controller").Do(ctx)
	s.NoError(err)
}

// TestFailure tests a test failure
func (s *ChartTestSuite) TestFailure(ctx context.Context) {
	time.Sleep(10 * time.Second)
	s.Fail("test failed")
}

// TestRemoteInstall tests a remote chart installation
func (s *ChartTestSuite) TestRemoteInstall(ctx context.Context) {
	err := s.Helm().Install("redis", "redis").
		RepoURL("https://charts.bitnami.com/bitnami").
		Set("architecture", "standalone").
		Set("auth.enabled", false).
		Wait().
		Do(ctx)
	s.NoError(err)

	err = s.Helm().
		Uninstall("redis").
		Do(ctx)
	s.NoError(err)
}
