// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package benchmark

import "github.com/onosproject/helmit/pkg/registry"

// Register registers a benchmark suite
// Deprecated: Use registry.RegisterBenchmarkSuite instead
func Register(name string, suite BenchmarkingSuite) {
	registry.RegisterBenchmarkSuite(name, suite)
}
