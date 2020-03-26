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

package simulation

import (
	"context"
	"fmt"
	atomix "github.com/atomix/go-client/pkg/client"
	"github.com/atomix/go-client/pkg/client/map"
	"github.com/onosproject/helmet/pkg/benchmark"
	"github.com/onosproject/helmet/pkg/helm"
	"github.com/onosproject/helmet/pkg/input"
	"github.com/onosproject/helmet/pkg/kubernetes"
	"github.com/onosproject/helmet/pkg/simulation"
	"github.com/onosproject/helmet/pkg/test"
	"time"
)

var keys = input.RandomChoice(input.SetOf(input.RandomString(8), 1000))
var values = input.RandomBytes(128)

// AtomixSimulationSuite is an end-to-end simulation suite for Atomix
type AtomixSimulationSuite struct {
	test.Suite
	m _map.Map
}

// ScheduleSimulator schedules simulator functions
func (s *AtomixSimulationSuite) ScheduleSimulator(sim *simulation.Simulator) {
	sim.Schedule("get", s.SimulateMapGet, 1*time.Second, 1)
	sim.Schedule("put", s.SimulateMapPut, 5*time.Second, 1)
	sim.Schedule("remove", s.SimulateMapRemove, 30*time.Second, 1)
}

// SetupSimulationSuite sets up the Atomix cluster
func (s *AtomixSimulationSuite) SetupSimulationSuite() error {
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
		Set("backend.image", "atomix/raft-replica:latest").
		Install(true)
	if err != nil {
		return err
	}
	return nil
}

func (s *AtomixSimulationSuite) getController() (string, error) {
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

// SetupSimulator creates an instance of the map on each simulator pod
func (s *AtomixSimulationSuite) SetupSimulator(c *benchmark.Context) error {
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

// SimulateMapPut simulates an Atomix map put operation
func (s *AtomixSimulationSuite) SimulateMapPut(c *simulation.Simulator) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_, err := s.m.Put(ctx, keys.Next().String(), values.Next().Bytes())
	return err
}

// SimulateMapPut simulates an Atomix map get operation
func (s *AtomixSimulationSuite) SimulateMapGet(c *simulation.Simulator) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_, err := s.m.Get(ctx, keys.Next().String())
	return err
}

// SimulateMapPut simulates an Atomix map remove operation
func (s *AtomixSimulationSuite) SimulateMapRemove(c *simulation.Simulator) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_, err := s.m.Remove(ctx, keys.Next().String())
	return err
}
