// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"github.com/onosproject/helmit/pkg/benchmark"
	tests "github.com/onosproject/helmit/test"
)

func main() {
	benchmark.Main(map[string]benchmark.BenchmarkingSuite{
		"chart": new(tests.ChartBenchmarkSuite),
	})
}
