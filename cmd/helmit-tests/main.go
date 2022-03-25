// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"github.com/onosproject/helmit/pkg/benchmark"
	"github.com/onosproject/helmit/pkg/registry"
	"github.com/onosproject/helmit/pkg/simulation"
	"github.com/onosproject/helmit/pkg/test"
	tests "github.com/onosproject/helmit/test"
	"os"
)

func main() {
	jobType := os.Getenv("JOB_TYPE")
	switch jobType {
	case "test":
		registry.RegisterTestSuite("chart", &tests.ChartTestSuite{})
		test.Main()
	case "benchmark":
		registry.RegisterBenchmarkSuite("chart", &tests.ChartBenchmarkSuite{})
		benchmark.Main()
	case "simulation":
		registry.RegisterSimulationSuite("chart", &tests.ChartSimulationSuite{})
		simulation.Main()
	}
}
