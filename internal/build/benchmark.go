// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package build

import (
	"github.com/onosproject/helmit/internal/logging"
	"github.com/onosproject/helmit/pkg/benchmark"
	"reflect"
)

const benchmarkMainTpl = `
package main

import (
	"github.com/onosproject/helmit/pkg/benchmark"
	{{- range .Imports }}
	{{ .Alias }} "{{ .Path }}"
	{{- end }}
)

func main() {
	benchmark.Main([]benchmark.BenchmarkingSuite{
		{{- range .Suites }}
		new({{ .Import.Alias }}.{{ .Name }}),
		{{- end }}
	})
}
`

const defaultBenchmarkSuiteMatcher = "BenchmarkSuite$"

// Benchmarks returns a new benchmark binary builder
func Benchmarks(log logging.Logger, suiteMatchers ...string) *Builder {
	if len(suiteMatchers) == 0 {
		suiteMatchers = []string{defaultBenchmarkSuiteMatcher}
	}
	return newBuilder(reflect.TypeOf(benchmark.Suite{}), suiteMatchers, benchmarkMainTpl, log)
}
