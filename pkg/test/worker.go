// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package test

import (
	"context"
	"fmt"
	"github.com/onosproject/helmit/pkg/helm"
	"github.com/onosproject/helmit/pkg/registry"
	"google.golang.org/grpc"
	"net"
	"os"
	"testing"
)

// newWorker returns a new test worker
func newWorker(config *Config) (*Worker, error) {
	return &Worker{
		config: config,
	}, nil
}

// Worker runs a test job
type Worker struct {
	config *Config
}

// Run runs a benchmark
func (w *Worker) Run() error {
	err := helm.SetContext(&helm.Context{
		WorkDir:    w.config.Context,
		Values:     w.config.Values,
		ValueFiles: w.config.ValueFiles,
	})
	if err != nil {
		return err
	}

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
	test := registry.GetTestSuite(request.Suite)
	if test == nil {
		fmt.Println(fmt.Errorf("unknown test suite %s", request.Suite))
		os.Exit(1)
	}

	tests := []testing.InternalTest{
		{
			Name: request.Suite,
			F: func(t *testing.T) {
				RunTests(t, test, request)
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
