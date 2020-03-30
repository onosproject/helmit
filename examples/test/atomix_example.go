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

package test

import (
	"bytes"
	"context"
	atomix "github.com/atomix/go-client/pkg/client"
	"github.com/atomix/go-client/pkg/client/map"
	"github.com/onosproject/helmit/pkg/helm"
	"github.com/onosproject/helmit/pkg/test"
	"github.com/stretchr/testify/assert"
	"testing"
)

// AtomixTestSuite is an end-to-end test suite for Atomix
type AtomixTestSuite struct {
	test.Suite
}

// SetupTestSuite sets up the Atomix cluster
func (s *AtomixTestSuite) SetupTestSuite() error {
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

// TestMap tests Atomix map operations
func (s *AtomixTestSuite) TestMap(t *testing.T) {
	client, err := atomix.New(
		"atomix-controller:5679",
		atomix.WithNamespace(helm.Namespace()),
		atomix.WithScope("test"))
	assert.NoError(t, err)

	database, err := client.GetDatabase(context.Background(), "atomix-raft")
	assert.NoError(t, err)

	m, err := database.GetMap(context.Background(), "TestMap")
	assert.NoError(t, err)

	ch := make(chan *_map.Entry)
	err = m.Entries(context.Background(), ch)
	assert.NoError(t, err)
	for range ch {
		assert.Fail(t, "entries found in map")
	}

	size, err := m.Len(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, 0, size)

	value, err := m.Get(context.Background(), "foo")
	assert.NoError(t, err)
	assert.Nil(t, value)

	value, err = m.Put(context.Background(), "foo", []byte("Hello world!"))
	assert.NoError(t, err)
	assert.NotNil(t, value)
	assert.Equal(t, "foo", value.Key)
	assert.True(t, bytes.Equal([]byte("Hello world!"), value.Value))
	assert.NotEqual(t, int64(0), value.Version)
	version := value.Version

	value, err = m.Get(context.Background(), "foo")
	assert.NoError(t, err)
	assert.NotNil(t, value)
	assert.Equal(t, "foo", value.Key)
	assert.True(t, bytes.Equal([]byte("Hello world!"), value.Value))
	assert.Equal(t, version, value.Version)

	size, err = m.Len(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, 1, size)

	ch = make(chan *_map.Entry)
	err = m.Entries(context.Background(), ch)
	assert.NoError(t, err)
	i := 0
	for kv := range ch {
		assert.Equal(t, "foo", kv.Key)
		assert.Equal(t, "Hello world!", string(kv.Value))
		i++
	}
	assert.Equal(t, 1, i)

	allEvents := make(chan *_map.Event)
	err = m.Watch(context.Background(), allEvents, _map.WithReplay())
	assert.NoError(t, err)

	event := <-allEvents
	assert.NotNil(t, event)
	assert.Equal(t, "foo", event.Entry.Key)
	assert.Equal(t, []byte("Hello world!"), event.Entry.Value)
	assert.Equal(t, value.Version, event.Entry.Version)

	futureEvents := make(chan *_map.Event)
	err = m.Watch(context.Background(), futureEvents)
	assert.NoError(t, err)

	value, err = m.Put(context.Background(), "bar", []byte("Hello world!"))
	assert.NoError(t, err)
	assert.NotNil(t, value)
	assert.Equal(t, "bar", value.Key)
	assert.Equal(t, []byte("Hello world!"), value.Value)
	assert.NotEqual(t, int64(0), value.Version)

	event = <-allEvents
	assert.NotNil(t, event)
	assert.Equal(t, "bar", event.Entry.Key)
	assert.Equal(t, []byte("Hello world!"), event.Entry.Value)
	assert.Equal(t, value.Version, event.Entry.Version)

	event = <-futureEvents
	assert.NotNil(t, event)
	assert.Equal(t, "bar", event.Entry.Key)
	assert.Equal(t, []byte("Hello world!"), event.Entry.Value)
	assert.Equal(t, value.Version, event.Entry.Version)
}
