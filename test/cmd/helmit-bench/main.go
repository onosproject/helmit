// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"github.com/onosproject/helmit/pkg/bench"
	tests "github.com/onosproject/helmit/test"
)

func main() {
	bench.Main(map[string]bench.BenchmarkingSuite{
		"chart": &tests.ChartBenchmarkSuite{},
	})
}
