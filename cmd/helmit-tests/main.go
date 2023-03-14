// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"github.com/onosproject/helmit/pkg/benchmark"
	"github.com/onosproject/helmit/pkg/simulation"
	"github.com/onosproject/helmit/pkg/test"
	tests "github.com/onosproject/helmit/test"
	"os"
)

func main() {
	jobType := os.Getenv("JOB_TYPE")
	switch jobType {
	case "test":
		test.Main(map[string]test.TestingSuite{
			"chart": &tests.ChartTestSuite{},
		})
	case "benchmark":
		benchmark.Main(map[string]benchmark.BenchmarkingSuite{
			"chart": &tests.ChartBenchmarkSuite{},
		})
	case "simulation":
		simulation.Main(map[string]simulation.SimulatingSuite{
			"chart": &tests.ChartSimulationSuite{},
		})
	}
}
