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

package benchmark

import (
	"context"
	"fmt"
	atomix "github.com/atomix/go-client/pkg/client"
	"github.com/atomix/go-client/pkg/client/map"
	"github.com/onosproject/helmet/pkg/benchmark"
	"github.com/onosproject/helmet/pkg/helm"
	"github.com/onosproject/helmet/pkg/input"
	"github.com/onosproject/helmet/pkg/kubernetes"
	"time"
)

var keys = input.RandomChoice(input.SetOf(input.RandomString(8), 1000))
var values = input.RandomBytes(128)

// AtomixBenchmarkSuite is an end-to-end test suite for Atomix
type AtomixBenchmarkSuite struct {
	benchmark.Suite
	m _map.Map
}

// SetupBenchmarkSuite sets up the Atomix cluster
func (s *AtomixBenchmarkSuite) SetupBenchmarkSuite() error {
	err := helm.Chart("atomix-controller").
		Release("atomix-controller").
		Set("scope", "Namespace").
		Install(true)
	if err != nil {
		return err
	}

	err = helm.Chart("atomix-database").
		Release("atomix-raft").
		Set("clusters", 3).
		Set("partitions", 10).
		Set("backend.replicas", 3).
		Set("backend.image", "atomix/raft-replica:latest").
		Install(true)
	if err != nil {
		return err
	}
	return nil
}

func (s *AtomixBenchmarkSuite) getController() (string, error) {
	client, err := kubernetes.NewForRelease(helm.Release("atomix-controller"))
	if err != nil {
		return "", err
	}
	services, err := client.CoreV1().Services().List()
	if err != nil {
		return "", err
	}
	if len(services) == 0 {
		return "", nil
	}
	service := services[0]
	return fmt.Sprintf("%s.%s.svc.cluster.local:%d", service.Name, service.Namespace, service.Ports()[0].Port), nil
}

// SetupBenchmarkWorker creates an instance of the map on each worker node
func (s *AtomixBenchmarkSuite) SetupBenchmarkWorker(c *benchmark.Context) error {
	address, err := s.getController()
	if err != nil {
		return err
	}

	client, err := atomix.New(address)
	if err != nil {
		return err
	}

	database, err := client.GetDatabase(context.Background(), "atomix-raft")
	if err != nil {
		return err
	}

	m, err := database.GetMap(context.Background(), "TestMap")
	if err != nil {
		return err
	}
	s.m = m
	return nil
}

// BenchmarkMapPut benchmarks an Atomix map put operation
func (s *AtomixBenchmarkSuite) BenchmarkMapPut(b *benchmark.Benchmark) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_, err := s.m.Put(ctx, keys.Next().String(), values.Next().Bytes())
	return err
}

// BenchmarkMapPut benchmarks an Atomix map get operation
func (s *AtomixBenchmarkSuite) BenchmarkMapGet(b *benchmark.Benchmark) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_, err := s.m.Get(ctx, keys.Next().String())
	return err
}

// BenchmarkMapPut benchmarks an Atomix map remove operation
func (s *AtomixBenchmarkSuite) BenchmarkMapRemove(b *benchmark.Benchmark) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_, err := s.m.Remove(ctx, keys.Next().String())
	return err
}
