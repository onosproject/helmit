// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package test

import (
	"github.com/onosproject/helmit/pkg/test"
)

// ChartTestSuite is a test for chart deployment
type ChartTestSuite struct {
	test.Suite
}

// TestLocalInstall tests a local chart installation
func (s *ChartTestSuite) TestLocalInstall() {
	err := s.Helm().Install("atomix-controller", "./controller/chart").
		Set("image.tag", "latest").
		Set("init.image.tag", "latest").
		Wait().
		Do(s.Context())
	s.NoError(err)

	err = s.Helm().Uninstall("atomix-controller").Do(s.Context())
	s.NoError(err)
}

// TestRemoteInstall tests a remote chart installation
func (s *ChartTestSuite) TestRemoteInstall() {
	err := s.Helm().Install("kafka", "http://storage.googleapis.com/kubernetes-charts-incubator").
		Set("replicas", 1).
		Set("zookeeper.replicaCount", 1).
		Wait().
		Do(s.Context())
	s.NoError(err)

	err = s.Helm().
		Uninstall("kafka").
		Do(s.Context())
	s.NoError(err)
}
