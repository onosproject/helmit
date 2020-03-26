// Copyright 2020-present Open Networking Foundation.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package test

import (
	"github.com/onosproject/helmet/pkg/benchmark"
	"github.com/onosproject/helmet/pkg/helm"
	"github.com/onosproject/helmet/pkg/input"
	"time"
)

// ChartBenchmarkSuite benchmarks a Helm chart
type ChartBenchmarkSuite struct {
	benchmark.Suite
	value input.Source
}

// SetupBenchmark :: benchmark
func (s *ChartBenchmarkSuite) SetupBenchmark(b *benchmark.Context) error {
	return helm.Chart("atomix-controller").
		Release("atomix-controller").
		Set("scope", "Namespace").
		Install(true)
}

// SetupWorker :: benchmark
func (s *ChartBenchmarkSuite) SetupWorker(b *benchmark.Context) error {
	s.value = input.RandomString(8)
	return nil
}

// BenchmarkTest :: benchmark
func (s *ChartBenchmarkSuite) BenchmarkTest(b *benchmark.Benchmark) error {
	println(s.value.Next().String())
	time.Sleep(time.Second)
	return nil
}
