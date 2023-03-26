// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package test

import (
	"context"
	"github.com/brianvoe/gofakeit/v6"
	"github.com/onosproject/helmit/pkg/benchmark"
	"math/rand"
	"time"
)

// ChartBenchmarkSuite benchmarks a Helm chart
type ChartBenchmarkSuite struct {
	benchmark.Suite
}

func (s *ChartBenchmarkSuite) SetupSuite(ctx context.Context) error {
	err := s.Helm().Install("atomix-controller", "./controller/chart").
		Wait().
		Do(ctx)
	if err != nil {
		return err
	}
	return nil
}

func (s *ChartBenchmarkSuite) BenchmarkFoo(ctx context.Context) error {
	println(gofakeit.Animal())
	time.Sleep(time.Duration(rand.Intn(250)) * time.Millisecond)
	return nil
}

func (s *ChartBenchmarkSuite) BenchmarkBar(ctx context.Context) error {
	println(gofakeit.Animal())
	time.Sleep(time.Duration(rand.Intn(500)) * time.Millisecond)
	return nil
}

func (s *ChartBenchmarkSuite) BenchmarkBaz(ctx context.Context) error {
	println(gofakeit.Animal())
	time.Sleep(time.Duration(rand.Intn(1000)) * time.Millisecond)
	return nil
}

func (s *ChartBenchmarkSuite) TearDownSuite(ctx context.Context) error {
	err := s.Helm().Uninstall("atomix-controller").
		Wait().
		Do(ctx)
	if err != nil {
		return err
	}
	return err
}
