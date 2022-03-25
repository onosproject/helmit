// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package test

import (
	"time"

	"github.com/onosproject/helmit/pkg/benchmark"
	"github.com/onosproject/helmit/pkg/helm"
	"github.com/onosproject/helmit/pkg/input"
)

// ChartBenchmarkSuite benchmarks a Helm chart
type ChartBenchmarkSuite struct {
	benchmark.Suite
	value input.Source
}

// SetupSuite :: benchmark
func (s *ChartBenchmarkSuite) SetupSuite(b *input.Context) error {
	atomix := helm.Chart("kubernetes-controller").
		Release("atomix-controller").
		Set("scope", "Namespace")

	err := atomix.Install(true)
	if err != nil {
		return err
	}

	err = atomix.Uninstall()
	if err != nil {
		return err
	}
	return nil

}

// SetupWorker :: benchmark
func (s *ChartBenchmarkSuite) SetupWorker(b *input.Context) error {
	s.value = input.RandomString(8)
	return nil
}

// BenchmarkTest :: benchmark
func (s *ChartBenchmarkSuite) BenchmarkTest(b *benchmark.Benchmark) error {
	println(s.value.Next().String())
	time.Sleep(time.Second)
	return nil
}
