// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package test

import (
	"time"

	"github.com/onosproject/helmit/pkg/simulation"
)

// ChartSimulationSuite :: simulation
type ChartSimulationSuite struct {
	simulation.Suite
}

func (s *ChartSimulationSuite) SetupSimulation() error {
	err := s.Helm().Install("atomix-controller", "./controller/chart").
		Wait().
		Do(s.Context())
	if err != nil {
		return err
	}
	return nil
}

// ScheduleSimulator :: simulation
func (s *ChartSimulationSuite) ScheduleSimulator() {
	s.S().Schedule("foo", s.SimulateFoo, 1*time.Second, 1)
	s.S().Schedule("bar", s.SimulateBar, 5*time.Second, 1)
	s.S().Schedule("baz", s.SimulateBaz, 30*time.Second, 1)
}

// SimulateFoo :: simulation
func (s *ChartSimulationSuite) SimulateFoo(sim *simulation.S) error {
	println(s.Context().Value("foo"))
	return nil
}

// SimulateBar :: simulation
func (s *ChartSimulationSuite) SimulateBar(sim *simulation.S) error {
	println(s.Context().Value("bar"))
	return nil
}

// SimulateBaz :: simulation
func (s *ChartSimulationSuite) SimulateBaz(sim *simulation.S) error {
	println(s.Context().Value("baz"))
	return nil
}

func (s *ChartSimulationSuite) TearDownSimulation() error {
	err := s.Helm().Uninstall("atomix-controller").
		Wait().
		Do(s.Context())
	if err != nil {
		return err
	}
	return err
}
