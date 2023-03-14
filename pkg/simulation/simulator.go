// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package simulation

import (
	"context"
	"fmt"
	"github.com/onosproject/helmit/pkg/helm"
	"github.com/onosproject/helmit/pkg/util/logging"
	"math/rand"
	"sync"
	"time"
)

// SimulatingSuite is a suite of simulators
type SimulatingSuite interface {
	SetS(s *S)
	S() *S
	SetHelm(helm *helm.Helm)
	Helm() *helm.Helm
	SetContext(ctx context.Context)
	Context() context.Context
}

// Suite is the base for a benchmark suite
type Suite struct {
	s    *S
	helm *helm.Helm
	ctx  context.Context
}

func (suite *Suite) Namespace() string {
	return suite.helm.Namespace()
}

func (suite *Suite) SetS(s *S) {
	suite.s = s
}

func (suite *Suite) S() *S {
	return suite.s
}

func (suite *Suite) SetHelm(helm *helm.Helm) {
	suite.helm = helm
}

func (suite *Suite) Helm() *helm.Helm {
	return suite.helm
}

func (suite *Suite) SetContext(ctx context.Context) {
	suite.ctx = ctx
}

func (suite *Suite) Context() context.Context {
	return suite.ctx
}

// ScheduleSimulator is an interface for scheduling operations for a simulation
type ScheduleSimulator interface {
	ScheduleSimulator()
}

// SetupSimulation is an interface for setting up a suite of simulators
type SetupSimulation interface {
	SetupSimulation() error
}

// TearDownSimulation is an interface for tearing down a suite of simulators
type TearDownSimulation interface {
	TearDownSimulation() error
}

// SetupSimulator is an interface for executing code before every simulator
type SetupSimulator interface {
	SetupSimulator() error
}

// TearDownSimulator is an interface for executing code after every simulator
type TearDownSimulator interface {
	TearDownSimulator() error
}

// newSimulator returns a new simulation instance
func newSimulator(name string, process int, suite SimulatingSuite, config *Config) *S {
	return &S{
		Name:    name,
		Process: process,
		suite:   suite,
		config:  config,
		ops:     make(map[string]*operation),
	}
}

// S is a simulator runner
type S struct {
	// Name is the name of the simulation
	Name string
	// Process is the unique identifier of the simulator process
	Process int
	config  *Config
	suite   SimulatingSuite
	ops     map[string]*operation
	mu      sync.Mutex
}

// Schedule schedules an operation
func (s *S) Schedule(name string, f func(*S) error, rate time.Duration, jitter float64) {
	if override, ok := s.config.Rates[name]; ok {
		rate = override
	}
	if override, ok := s.config.Jitter[name]; ok {
		jitter = override
	}
	s.ops[name] = &operation{
		name:      name,
		f:         f,
		rate:      rate,
		jitter:    jitter,
		simulator: s,
		stopCh:    make(chan error),
	}
}

// lock locks the simulation
func (s *S) lock() {
	s.mu.Lock()
}

// unlock unlocks the simulator
func (s *S) unlock() {
	s.mu.Unlock()
}

// setupSimulation sets up the simulation
func (s *S) setupSimulation() error {
	if setupSuite, ok := s.suite.(SetupSimulation); ok {
		return setupSuite.SetupSimulation()
	}
	return nil
}

// teardownSimulation tears down the simulation
func (s *S) teardownSimulation() error {
	if tearDownSuite, ok := s.suite.(TearDownSimulation); ok {
		return tearDownSuite.TearDownSimulation()
	}
	return nil
}

// setupSimulator sets up the simulator
func (s *S) setupSimulator() error {
	if setupSuite, ok := s.suite.(SetupSimulator); ok {
		if err := setupSuite.SetupSimulator(); err != nil {
			return err
		}
	}
	if setupSuite, ok := s.suite.(ScheduleSimulator); ok {
		setupSuite.ScheduleSimulator()
	}
	return nil
}

// teardownSimulator tears down the simulator
func (s *S) teardownSimulator() error {
	if tearDownSuite, ok := s.suite.(TearDownSimulator); ok {
		return tearDownSuite.TearDownSimulator()
	}
	return nil
}

// start starts the simulator
func (s *S) start() {
	for _, op := range s.ops {
		go op.start()
	}
}

// stop stops the simulator
func (s *S) stop() {
	for _, op := range s.ops {
		op.stop()
	}
}

// waitJitter returns a channel that closes after time.Duration between duration and duration + maxFactor *
// duration.
func waitJitter(duration time.Duration, maxFactor float64) <-chan time.Time {
	if maxFactor <= 0.0 {
		maxFactor = 1.0
	}
	delay := duration + time.Duration(rand.Float64()*maxFactor*float64(duration))
	return time.After(delay)
}

// operation is a simulator operation
type operation struct {
	name      string
	f         func(*S) error
	rate      time.Duration
	jitter    float64
	simulator *S
	stopCh    chan error
}

// start starts the operation simulator
func (o *operation) start() {
	for {
		select {
		case <-waitJitter(o.rate, o.jitter):
			o.simulator.lock()
			o.run()
			o.simulator.unlock()
		case <-o.stopCh:
			return
		}
	}
}

// run runs the operation
func (o *operation) run() {
	step := logging.NewStep(fmt.Sprintf("%s/%d", o.simulator.Name, getSimulatorID()), "Run %s", o.name)
	step.Start()
	if err := o.f(o.simulator); err != nil {
		step.Fail(err)
	} else {
		step.Complete()
	}
}

// stop stops the operation simulator
func (o *operation) stop() {
	close(o.stopCh)
}
