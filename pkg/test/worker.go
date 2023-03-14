// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package test

import (
	"context"
	"fmt"
	"github.com/onosproject/helmit/pkg/helm"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
	"net"
	"os"
	"reflect"
	"regexp"
	"runtime/debug"
	"testing"
)

// newWorker returns a new test worker
func newWorker(suites map[string]TestingSuite, config *Config) (*Worker, error) {
	return &Worker{
		suites: suites,
		config: config,
	}, nil
}

// Worker runs a test job
type Worker struct {
	suites map[string]TestingSuite
	config *Config
}

// Run runs a test
func (w *Worker) Run() error {
	lis, err := net.Listen("tcp", ":5000")
	if err != nil {
		return err
	}
	server := grpc.NewServer()
	RegisterWorkerServiceServer(server, w)
	return server.Serve(lis)
}

// RunTests runs a suite of tests
func (w *Worker) RunTests(ctx context.Context, request *TestRequest) (*TestResponse, error) {
	go w.runTests(request)
	return &TestResponse{}, nil
}

func (w *Worker) runTests(request *TestRequest) {
	test, ok := w.suites[request.Suite]
	if !ok {
		fmt.Println(fmt.Errorf("unknown test suite %s", request.Suite))
		os.Exit(1)
	}

	tests := []testing.InternalTest{
		{
			Name: request.Suite,
			F: func(t *testing.T) {
				w.runSuite(t, test, request)
			},
		},
	}

	// Hack to enable verbose testing.
	os.Args = []string{
		os.Args[0],
		"-test.v",
	}

	testing.Main(func(_, _ string) (bool, error) { return true, nil }, tests, nil, nil)
}

// RunTests runs a test suite
func (w *Worker) runSuite(t *testing.T, s TestingSuite, request *TestRequest) {
	defer failTestOnPanic(t)

	parentCtx := context.Background()
	for key, value := range request.Args {
		parentCtx = context.WithValue(parentCtx, key, value)
	}

	s.SetT(t)
	s.SetContext(parentCtx)
	s.SetHelm(helm.NewClient(helm.Context{
		Namespace:  w.config.Namespace,
		WorkDir:    w.config.Context,
		Values:     w.config.Values,
		ValueFiles: w.config.ValueFiles,
	}))

	var suiteSetupDone bool

	methodFinder := reflect.TypeOf(s)
	tests := []testing.InternalTest{}
	for index := 0; index < methodFinder.NumMethod(); index++ {
		method := methodFinder.Method(index)
		ok, err := testFilter(method.Name, request.Tests)
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid regexp for -m: %s\n", err)
			os.Exit(1)
		}
		if !ok {
			continue
		}
		if !suiteSetupDone {
			ctx, cancel := context.WithTimeout(parentCtx, w.config.Timeout)
			s.SetContext(ctx)
			if setupTestSuite, ok := s.(suite.SetupAllSuite); ok {
				setupTestSuite.SetupSuite()
			}
			if setupTestSuite, ok := s.(SetupTestSuite); ok {
				if err := setupTestSuite.SetupTestSuite(); err != nil {
					panic(err)
				}
			}
			s.SetContext(parentCtx)
			cancel()
			suiteSetupDone = true
		}

		test := testing.InternalTest{
			Name: method.Name,
			F: func(t *testing.T) {
				defer failTestOnPanic(t)

				parentT := s.T()
				s.SetT(t)

				ctx, cancel := context.WithTimeout(parentCtx, w.config.Timeout)
				defer cancel()
				s.SetContext(ctx)

				if setupTest, ok := s.(suite.SetupTestSuite); ok {
					setupTest.SetupTest()
				}
				if setupTest, ok := s.(SetupTest); ok {
					if err := setupTest.SetupTest(); err != nil {
						panic(err)
					}
				}
				if beforeTest, ok := s.(suite.BeforeTest); ok {
					beforeTest.BeforeTest(methodFinder.Elem().Name(), method.Name)
				}
				if beforeTest, ok := s.(BeforeTest); ok {
					if err := beforeTest.BeforeTest(method.Name); err != nil {
						panic(err)
					}
				}
				defer func() {
					if afterTest, ok := s.(suite.AfterTest); ok {
						afterTest.AfterTest(methodFinder.Elem().Name(), method.Name)
					}
					if afterTest, ok := s.(AfterTest); ok {
						if err := afterTest.AfterTest(method.Name); err != nil {
							panic(err)
						}
					}
					if tearDownTest, ok := s.(suite.TearDownTestSuite); ok {
						tearDownTest.TearDownTest()
					}
					if tearDownTest, ok := s.(TearDownTest); ok {
						if err := tearDownTest.TearDownTest(); err != nil {
							panic(err)
						}
					}
					s.SetContext(parentCtx)
					s.SetT(parentT)
				}()
				method.Func.Call([]reflect.Value{reflect.ValueOf(s)})
			},
		}
		tests = append(tests, test)
	}

	if suiteSetupDone {
		defer func() {
			ctx, cancel := context.WithTimeout(parentCtx, w.config.Timeout)
			s.SetContext(ctx)
			if tearDownTestSuite, ok := s.(suite.TearDownAllSuite); ok {
				tearDownTestSuite.TearDownSuite()
			}
			if tearDownTestSuite, ok := s.(TearDownTestSuite); ok {
				if err := tearDownTestSuite.TearDownTestSuite(); err != nil {
					panic(err)
				}
			}
			s.SetContext(parentCtx)
			cancel()
		}()
	}

	runTests(t, tests)
}

// runSuite runs a test
func runTests(t *testing.T, tests []testing.InternalTest) {
	for _, test := range tests {
		t.Run(test.Name, test.F)
	}
}

// testFilter filters test method names
func testFilter(name string, cases []string) (bool, error) {
	if ok, _ := regexp.MatchString("^Test", name); !ok {
		return false, nil
	}

	if len(cases) == 0 || cases[0] == "" {
		return true, nil
	}

	for _, test := range cases {
		if test == name {
			return true, nil
		}
	}
	return false, nil
}

func failTestOnPanic(t *testing.T) {
	r := recover()
	if r != nil {
		t.Errorf("test panicked: %v\n%s", r, debug.Stack())
		t.FailNow()
	}
}
