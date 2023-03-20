// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package test

import (
	"context"
	"fmt"
	api "github.com/onosproject/helmit/api/v1"
	"github.com/onosproject/helmit/internal/job"
	"github.com/onosproject/helmit/internal/k8s"
	"github.com/onosproject/helmit/pkg/helm"
	"github.com/onosproject/onos-lib-go/pkg/errors"
	"google.golang.org/grpc"
	"net"
	"reflect"
	"strings"
	"testing"
)

// newWorker returns a new benchmark worker
func newWorker(spec job.Spec, suites map[string]TestingSuite, t *testing.T) (*testWorker, error) {
	return &testWorker{
		spec:   spec,
		suites: suites,
		t:      t,
	}, nil
}

// testWorker runs a benchmark job
type testWorker struct {
	spec   job.Spec
	suites map[string]TestingSuite
	t      *testing.T
}

// run runs a benchmark
func (w *testWorker) run() error {
	lis, err := net.Listen("tcp", ":5000")
	if err != nil {
		return err
	}
	server := grpc.NewServer()
	api.RegisterTesterServer(server, w)
	return server.Serve(lis)
}

// SetupTestSuite sets up a benchmark suite
func (w *testWorker) SetupTestSuite(ctx context.Context, request *api.SetupTestSuiteRequest) (*api.SetupTestSuiteResponse, error) {
	suite, ok := w.suites[request.Suite]
	if !ok {
		return nil, errors.NewNotFound("unknown suite %s", request.Suite)
	}

	config, err := k8s.GetConfig()
	if err != nil {
		return nil, err
	}
	suite.SetConfig(config)

	suite.SetHelm(helm.NewClient(helm.Context{
		Namespace:  w.spec.Namespace,
		WorkDir:    w.spec.Context,
		Values:     w.spec.Values,
		ValueFiles: w.spec.ValueFiles,
	}))

	if setupSuite, ok := suite.(SetupSuite); ok {
		ctx, cancel := context.WithTimeout(ctx, w.spec.Timeout)
		defer cancel()
		if err := setupSuite.SetupSuite(ctx); err != nil {
			return nil, err
		}
	}
	return &api.SetupTestSuiteResponse{}, nil
}

// TearDownTestSuite tears down a benchmark suite
func (w *testWorker) TearDownTestSuite(ctx context.Context, request *api.TearDownTestSuiteRequest) (*api.TearDownTestSuiteResponse, error) {
	suite, ok := w.suites[request.Suite]
	if !ok {
		return nil, errors.NewNotFound("unknown suite %s", request.Suite)
	}

	if tearDownSuite, ok := suite.(TearDownSuite); ok {
		ctx, cancel := context.WithTimeout(ctx, w.spec.Timeout)
		defer cancel()
		if err := tearDownSuite.TearDownSuite(ctx); err != nil {
			return nil, err
		}
	}
	return &api.TearDownTestSuiteResponse{}, nil
}

// SetupTest sets up a benchmark
func (w *testWorker) SetupTest(ctx context.Context, request *api.SetupTestRequest) (*api.SetupTestResponse, error) {
	suite, ok := w.suites[request.Suite]
	if !ok {
		return nil, errors.NewNotFound("unknown suite %s", request.Suite)
	}

	if setupTest, ok := suite.(SetupTest); ok {
		ctx, cancel := context.WithTimeout(ctx, w.spec.Timeout)
		defer cancel()
		if err := setupTest.SetupTest(ctx); err != nil {
			return nil, err
		}
	}

	methods := reflect.TypeOf(suite)
	if method, ok := methods.MethodByName("Setup" + request.Test); ok {
		method.Func.Call([]reflect.Value{reflect.ValueOf(suite)})
	}
	return &api.SetupTestResponse{}, nil
}

// TearDownTest tears down a benchmark
func (w *testWorker) TearDownTest(ctx context.Context, request *api.TearDownTestRequest) (*api.TearDownTestResponse, error) {
	suite, ok := w.suites[request.Suite]
	if !ok {
		return nil, errors.NewNotFound("unknown suite %s", request.Suite)
	}

	if tearDownTest, ok := suite.(TearDownTest); ok {
		ctx, cancel := context.WithTimeout(ctx, w.spec.Timeout)
		defer cancel()
		if err := tearDownTest.TearDownTest(ctx); err != nil {
			return nil, err
		}
	}

	methods := reflect.TypeOf(suite)
	if method, ok := methods.MethodByName("TearDown" + request.Test); ok {
		method.Func.Call([]reflect.Value{reflect.ValueOf(suite)})
	}
	return &api.TearDownTestResponse{}, nil
}

func (w *testWorker) GetTestSuites(ctx context.Context, request *api.GetTestSuitesRequest) (*api.GetTestSuitesResponse, error) {
	var suites []*api.TestSuite
	for name, suite := range w.suites {
		var tests []*api.Test
		methodFinder := reflect.TypeOf(suite)
		for index := 0; index < methodFinder.NumMethod(); index++ {
			method := methodFinder.Method(index)
			if strings.HasPrefix(method.Name, "Test") {
				name := method.Name[4:]
				_, setup := suite.(SetupTest)
				if !setup {
					_, setup = methodFinder.MethodByName(fmt.Sprintf("Setup%s", name))
				}
				_, teardown := suite.(TearDownTest)
				if !teardown {
					_, teardown = methodFinder.MethodByName(fmt.Sprintf("TearDown%s", name))
				}
				tests = append(tests, &api.Test{
					Name:     method.Name,
					Setup:    setup,
					TearDown: teardown,
				})
			}
		}
		_, setup := suite.(SetupSuite)
		_, teardown := suite.(TearDownSuite)
		suites = append(suites, &api.TestSuite{
			Name:     name,
			Setup:    setup,
			TearDown: teardown,
			Tests:    tests,
		})
	}
	return &api.GetTestSuitesResponse{
		Suites: suites,
	}, nil
}

func (w *testWorker) RunTest(ctx context.Context, request *api.RunTestRequest) (*api.RunTestResponse, error) {
	suite, ok := w.suites[request.Suite]
	if !ok {
		return nil, fmt.Errorf("unknown test suite %s", request.Suite)
	}
	methodFinder := reflect.TypeOf(suite)
	method, ok := methodFinder.MethodByName(request.Test)
	if !ok {
		return nil, fmt.Errorf("unknown test %s", request.Test)
	}
	succeeded := w.t.Run(request.Test, func(t *testing.T) {
		suite.SetT(t)
		method.Func.Call([]reflect.Value{reflect.ValueOf(suite), reflect.ValueOf(ctx)})
	})
	return &api.RunTestResponse{
		Succeeded: succeeded,
	}, nil
}
