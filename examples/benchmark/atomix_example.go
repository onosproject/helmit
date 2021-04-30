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
	atomix "github.com/atomix/go-client/pkg/client"
	"github.com/atomix/go-client/pkg/client/map"
	"github.com/onosproject/helmit/pkg/benchmark"
	"github.com/onosproject/helmit/pkg/helm"
	"github.com/onosproject/helmit/pkg/input"
	"github.com/onosproject/helmit/pkg/kubernetes"
	"github.com/onosproject/helmit/pkg/util"
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
func (s *AtomixBenchmarkSuite) SetupSuite(c *util.Context) error {
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

// SetupBenchmarkWorker creates an instance of the map on each worker node
func (s *AtomixBenchmarkSuite) SetupWorker(c *util.Context) error {
	address, err := getControllerAddress()
	if err != nil {
		return err
	}
	client, err := atomix.New(
		address,
		atomix.WithNamespace(helm.Namespace()),
		atomix.WithScope(c.Name))
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

// getControllerAddress returns the Atomix controller address
func getControllerAddress() (string, error) {
	release := helm.Chart("atomix-controller").Release("atomix-controller")
	client, err := kubernetes.NewForRelease(release)
	if err != nil {
		return "", err
	}
	services, err := client.CoreV1().Services().List()
	if err != nil || len(services) == 0 {
		return "", err
	}
	service := services[0]
	port := service.Ports()[0]
	return port.Address(false), nil
}
