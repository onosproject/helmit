// SPDX-FileCopyrightText: 2023-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package test

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestGetSuiteName(t *testing.T) {
	assert.Equal(t, "testSuite", getSuiteName(new(testSuite)))
	assert.Equal(t, "testSuite", getSuiteName(&testSuite{}))
	assert.Equal(t, "subTestSuite", getSuiteName(&subTestSuite{}))
}

func TestPatterns(t *testing.T) {
	assert.True(t, isRunnable("FooSuite", []string{}))
	assert.True(t, isRunnable("FooSuite", []string{"FooSuite"}))
	assert.True(t, isRunnable("FooSuite", []string{"Suite$"}))
	assert.True(t, isRunnable("FooSuite", []string{"Suite$", "NotSuite$"}))
	assert.True(t, isRunnable("FooSuite", []string{"Suite$/^Test"}))
	assert.True(t, isRunnable("FooSuite", []string{"Suite$/^Test", "NotSuite$"}))
	assert.False(t, isRunnable("FooSuite", []string{"NotSuite$"}))

	assert.True(t, isTestRunnable(t, "TestFoo", []string{}))
	assert.True(t, isTestRunnable(t, "TestFoo", []string{"TestPatterns"}))
	assert.True(t, isTestRunnable(t, "TestFoo", []string{"TestPatterns/TestFoo"}))
	assert.True(t, isTestRunnable(t, "TestFoo", []string{"^Test"}))
	assert.True(t, isTestRunnable(t, "TestFoo", []string{"^Test", "^Test/Foo"}))
	assert.True(t, isTestRunnable(t, "TestFoo", []string{"TestPatterns/^Test"}))
	assert.False(t, isTestRunnable(t, "TestFoo", []string{"TestFoo"}))
	assert.False(t, isTestRunnable(t, "TestFoo", []string{"TestBar"}))
}

func TestSuite(t *testing.T) {
	config := Config{
		Namespace: "foo",
		Tests: []string{
			"TestSuite/TestTest",
			"TestSuite/TestSubTest/Foo",
			"TestSuite/TestSubSuite/subTestSuite/TestSubTest/Bar",
		},
		Args: map[string]string{
			"bar": "baz",
		},
		Timeout: time.Minute,
	}
	secrets := map[string]string{
		"baz": "foo",
	}
	suite := &testSuite{}
	run(t, suite, config, secrets)

	assert.True(t, suite.setupSuite)
	assert.True(t, suite.setupTest)
	assert.True(t, suite.setupTestTest)
	assert.True(t, suite.testTest)
	assert.True(t, suite.tearDownTestTest)
	assert.True(t, suite.testSubTest)
	assert.True(t, suite.testSubTestFoo)
	assert.False(t, suite.testSubTestBar)
	assert.True(t, suite.testSubSuite)
	assert.True(t, suite.tearDownTest)
	assert.True(t, suite.tearDownSuite)
}

type testSuite struct {
	Suite
	setupSuite       bool
	setupTest        bool
	setupTestTest    bool
	tearDownTestTest bool
	testTest         bool
	testSubTest      bool
	testSubTestFoo   bool
	testSubTestBar   bool
	testSubSuite     bool
	tearDownTest     bool
	tearDownSuite    bool
}

func (t *testSuite) SetupSuite() {
	t.NotNil(t.T())
	t.NotNil(t.Context())
	t.Equal("foo", t.Namespace())
	t.Equal("baz", t.Arg("bar").String())
	t.Equal("foo", t.Secret("baz"))
	_, ok := t.Context().Deadline()
	t.False(ok)
	t.setupSuite = true
}

func (t *testSuite) SetupTest() {
	t.setupTest = true
}

func (t *testSuite) TearDownTest() {
	t.tearDownTest = true
}

func (t *testSuite) SetupTestTest() {
	t.setupTestTest = true
}

func (t *testSuite) TestTest() {
	_, ok := t.Context().Deadline()
	t.True(ok)
	t.testTest = true
}

func (t *testSuite) TearDownTestTest() {
	t.tearDownTestTest = true
}

func (t *testSuite) TestSubTest() {
	_, ok := t.Context().Deadline()
	t.True(ok)
	t.Run("Foo", func() {
		t.testSubTestFoo = true
	})
	t.Run("Bar", func() {
		t.testSubTestBar = true
	})
	t.testSubTest = true
}

func (t *testSuite) TestSubSuite() {
	_, ok := t.Context().Deadline()
	t.True(ok)
	subSuite := &subTestSuite{}
	t.True(t.RunSuite(subSuite))
	t.True(subSuite.setupSuite)
	t.True(subSuite.setupTest)
	t.False(subSuite.setupTestTest)
	t.False(subSuite.testTest)
	t.False(subSuite.tearDownTestTest)
	t.True(subSuite.testSubTest)
	t.False(subSuite.testSubTestFoo)
	t.True(subSuite.testSubTestBar)
	t.False(subSuite.testSubSuite)
	t.True(subSuite.tearDownTest)
	t.True(subSuite.tearDownSuite)
	t.testSubSuite = true
}

func (t *testSuite) TearDownSuite() {
	t.NotNil(t.T())
	t.NotNil(t.Context())
	t.Equal("foo", t.Namespace())
	t.Equal("baz", t.Arg("bar").String())
	t.Equal("foo", t.Secret("baz"))
	_, ok := t.Context().Deadline()
	t.False(ok)
	t.tearDownSuite = true
}

type subTestSuite struct {
	testSuite
}
