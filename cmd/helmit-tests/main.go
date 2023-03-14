// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"github.com/onosproject/helmit/pkg/benchmark"
	"github.com/onosproject/helmit/pkg/job"
	"github.com/onosproject/helmit/pkg/simulation"
	"github.com/onosproject/helmit/pkg/test"
	tests "github.com/onosproject/helmit/test"
)

func main() {
	switch job.GetType() {
	case job.TestType:
		test.Main(map[string]test.TestingSuite{
			"chart": &tests.ChartTestSuite{},
		})
	case job.BenchmarkType:
		benchmark.Main(map[string]benchmark.BenchmarkingSuite{
			"chart": &tests.ChartBenchmarkSuite{},
		})
	case job.SimulationType:
		simulation.Main(map[string]simulation.SimulatingSuite{
			"chart": &tests.ChartSimulationSuite{},
		})
	}
}
