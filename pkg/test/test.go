// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package test

import (
	"context"
	"github.com/onosproject/helmit/internal/k8s"
	"github.com/onosproject/helmit/pkg/helm"
	"github.com/onosproject/helmit/pkg/types"
	"github.com/stretchr/testify/suite"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"reflect"
	"regexp"
	"runtime/debug"
	"strings"
	"testing"
)

// TestingSuite is a suite of tests
type TestingSuite interface {
	suite.TestingSuite
	// Init initializes the suite
	Init(config Config, secrets map[string]string)
	// SetContext sets the test context
	SetContext(ctx context.Context)
	// Context returns the test context
	Context() context.Context
	// Namespace returns the suite namespace
	Namespace() string
	// Config returns the Kubernetes REST configuration
	Config() *rest.Config
	// Secret returns a secret by name
	Secret(name string) string
	// Secrets returns the injected secrets
	Secrets() map[string]string
	// Arg gets an argument by name
	Arg(name string) types.Value
	// Args returns a map of all test arguments
	Args() map[string]types.Value
	// Helm returns the Helm client
	Helm() *helm.Helm
	// Run runs a subtest
	Run(name string, f func()) bool
	// RunSuite runs a sub-suite
	RunSuite(suite TestingSuite) bool
}

// SetupSuite has a SetupSuite method, which will run before the
// tests in the suite are run.
type SetupSuite interface {
	// SetupSuite is called at the beginning of a test run to set up the test suite
	SetupSuite()
}

// TearDownSuite has a TearDownSuite method, which will run after
// all the tests in the suite have been run.
type TearDownSuite interface {
	// TearDownSuite is called at the end of a test run to tear down the test suite
	TearDownSuite()
}

// SetupTest has a SetupTest method, which will run before each
// test in the suite.
type SetupTest interface {
	// SetupTest is called at the beginning of each test run to set up the test
	SetupTest()
}

// TearDownTest has a TearDownTest method, which will run after
// each test in the suite.
type TearDownTest interface {
	// TearDownTest is called at the end of each test run to tear down the test
	TearDownTest()
}

// Suite is the base for a test suite
type Suite struct {
	suite.Suite
	*kubernetes.Clientset
	config     Config
	secrets    map[string]string
	restConfig *rest.Config
	helm       *helm.Helm
	args       map[string]types.Value
	ctx        context.Context
}

// Init initializes the test suite
func (suite *Suite) Init(config Config, secrets map[string]string) {
	suite.config = config
	suite.secrets = secrets

	args := make(map[string]types.Value)
	for key, value := range config.Args {
		args[key] = types.NewValue(value)
	}
	suite.args = args

	restConfig, err := k8s.GetConfig()
	suite.NoError(err)
	suite.restConfig = restConfig

	clientset, err := kubernetes.NewForConfig(restConfig)
	suite.NoError(err)
	suite.Clientset = clientset

	suite.helm = helm.NewClient(helm.Context{
		Namespace:  config.Namespace,
		WorkDir:    config.Context,
		Values:     config.Values,
		ValueFiles: config.ValueFiles,
	})
}

// SetContext sets the test context
func (suite *Suite) SetContext(ctx context.Context) {
	suite.ctx = ctx
}

// Context returns the test context
func (suite *Suite) Context() context.Context {
	return suite.ctx
}

// Namespace returns the suite namespace
func (suite *Suite) Namespace() string {
	return suite.config.Namespace
}

// SetConfig sets the Kubernetes REST configuration
func (suite *Suite) SetConfig(config *rest.Config) {
	suite.restConfig = config
	suite.Clientset = kubernetes.NewForConfigOrDie(config)
}

// Config returns the Kubernetes REST configuration
func (suite *Suite) Config() *rest.Config {
	restConfig, err := rest.InClusterConfig()
	if err != nil {
		suite.T().Fatal(err)
	}
	return restConfig
}

// SetHelm sets the Helm client
func (suite *Suite) SetHelm(helm *helm.Helm) {
	suite.helm = helm
}

// Helm returns the Helm client
func (suite *Suite) Helm() *helm.Helm {
	return suite.helm
}

// Secret returns a test secret by name
func (suite *Suite) Secret(name string) string {
	return suite.secrets[name]
}

// Secrets returns the injected secrets
func (suite *Suite) Secrets() map[string]string {
	return suite.secrets
}

// Arg returns a test argument by name
func (suite *Suite) Arg(name string) types.Value {
	value, ok := suite.args[name]
	if !ok {
		return types.NewValue(nil)
	}
	return value
}

// Args returns the test arguments
func (suite *Suite) Args() map[string]types.Value {
	return suite.args
}

// Run runs a test function
func (suite *Suite) Run(name string, subtest func()) bool {
	parentT := suite.T()
	if !isTestRunnable(parentT, name, suite.config.Tests) {
		return true
	}

	defer suite.SetT(parentT)
	parentCtx := suite.Context()
	defer suite.SetContext(parentCtx)
	return parentT.Run(name, func(t *testing.T) {
		suite.SetT(t)
		ctx, cancel := context.WithTimeout(context.Background(), suite.config.Timeout)
		defer cancel()
		suite.SetContext(ctx)
		subtest()
	})
}

// RunSuite runs a test suite
func (suite *Suite) RunSuite(subsuite TestingSuite) bool {
	name := getSuiteName(subsuite)
	if !isRunnable(name, suite.config.Suites) {
		return true
	}
	return suite.Run(name, func() {
		run(suite.T(), subsuite, suite.config, suite.secrets)
	})
}

var _ TestingSuite = (*Suite)(nil)

// run a test suite
func run(t *testing.T, suite TestingSuite, config Config, secrets map[string]string) {
	defer recoverAndFailOnPanic(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	suite.SetT(t)
	suite.SetContext(ctx)
	suite.Init(config, secrets)

	var suiteSetupDone bool

	methodFinder := reflect.TypeOf(suite)
	for i := 0; i < methodFinder.NumMethod(); i++ {
		method := methodFinder.Method(i)

		if !isRunnable(method.Name, config.Methods) {
			continue
		}
		if !isTestRunnable(t, method.Name, config.Tests) {
			continue
		}

		if !suiteSetupDone {
			if setupSuite, ok := suite.(SetupSuite); ok {
				setupSuite.SetupSuite()
			}
			suiteSetupDone = true
		}

		suite.Run(method.Name, func() {
			t := suite.T()
			defer recoverAndFailOnPanic(t)
			defer func() {
				r := recover()
				if tearDownMethod, ok := methodFinder.MethodByName("TearDown" + method.Name); ok {
					tearDownMethod.Func.Call([]reflect.Value{reflect.ValueOf(suite)})
				}
				if tearDownTest, ok := suite.(TearDownTest); ok {
					tearDownTest.TearDownTest()
				}
				failOnPanic(t, r)
			}()

			if setupTest, ok := suite.(SetupTest); ok {
				setupTest.SetupTest()
			}
			if setupMethod, ok := methodFinder.MethodByName("Setup" + method.Name); ok {
				setupMethod.Func.Call([]reflect.Value{reflect.ValueOf(suite)})
			}

			method.Func.Call([]reflect.Value{reflect.ValueOf(suite)})
		})
	}

	if suiteSetupDone && !config.NoTeardown {
		defer func() {
			if tearDownSuite, ok := suite.(TearDownSuite); ok {
				tearDownSuite.TearDownSuite()
			}
		}()
	}
}

func recoverAndFailOnPanic(t *testing.T) {
	r := recover()
	failOnPanic(t, r)
}

func failOnPanic(t *testing.T, r interface{}) {
	if r != nil {
		t.Errorf("test panicked: %v\n%s", r, debug.Stack())
		t.FailNow()
	}
}

func getSuiteName(suite TestingSuite) string {
	t := reflect.TypeOf(suite)
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return t.Name()
}

func isRunnable(name string, patterns []string) bool {
	if len(patterns) == 0 {
		return true
	}
	return matchesPatterns([]string{name}, patterns)
}

func isTestRunnable(t *testing.T, name string, patterns []string) bool {
	if len(patterns) == 0 {
		return true
	}
	names := append(strings.Split(t.Name(), "/"), name)
	return matchesPatterns(names, patterns)
}

func matchesPatterns(names []string, patterns []string) bool {
	for _, pattern := range patterns {
		if matchesPattern(names, pattern) {
			return true
		}
	}
	return false
}

func matchesPattern(names []string, pattern string) bool {
	patterns := strings.Split(pattern, "/")
	for i, name := range names {
		if i+1 > len(patterns) {
			break
		}
		if ok, _ := regexp.MatchString(patterns[i], name); !ok {
			return false
		}
	}
	return true
}
