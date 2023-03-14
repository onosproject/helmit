// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package test

import (
	"time"

	"github.com/onosproject/helmit/pkg/benchmark"
	"github.com/onosproject/helmit/pkg/input"
)

// ChartBenchmarkSuite benchmarks a Helm chart
type ChartBenchmarkSuite struct {
	benchmark.Suite
	value input.Source
}

func (s *ChartBenchmarkSuite) SetupSuite() error {
	err := s.Helm().Install("atomix-controller", "./controller/chart").
		Wait().
		Do(s.Context())
	if err != nil {
		return err
	}
	return nil
}

func (s *ChartBenchmarkSuite) SetupWorker() error {
	s.value = input.RandomString(8)
	return nil
}

func (s *ChartBenchmarkSuite) BenchmarkTest() error {
	println(s.value.Next().String())
	time.Sleep(time.Second)
	return nil
}

func (s *ChartBenchmarkSuite) TearDownSuite() error {
	err := s.Helm().Uninstall("atomix-controller").
		Wait().
		Do(s.Context())
	if err != nil {
		return err
	}
	return err
}
