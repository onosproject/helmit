// SPDX-FileCopyrightText: 2023-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package helm

import (
	"github.com/onosproject/helmit/pkg/types"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestReleaseValues(t *testing.T) {
	// a: b
	// b:
	//   c: 1
	// d:
	//   e: foo
	//   f: bar
	context := Context{
		Values:     map[string][]string{},
		ValueFiles: map[string][]string{},
	}
	defaultValues := map[string]any{
		"a": "b",
		"b": map[string]any{
			"c": 1,
		},
	}
	defaultFiles := []string{
		"context_test_defaults.yaml",
	}
	values, err := context.getReleaseValues("foo", defaultValues, defaultFiles)
	assert.NoError(t, err)
	assert.Equal(t, "b", values["a"])
	assert.Equal(t, 1, types.NewValue(values["b"].(map[string]any)["c"]).Int())
	assert.Equal(t, "foo", values["d"].(map[string]any)["e"])
	assert.Equal(t, "bar", values["d"].(map[string]any)["f"])

	// a: c
	// b:
	//   c: 3
	// d:
	//   e: foo
	//   f: baz
	context = Context{
		Values: map[string][]string{
			"foo": {
				"a=c",
				"b.c=3",
			},
		},
		ValueFiles: map[string][]string{
			"foo": {
				"context_test_overrides.yaml",
			},
		},
	}
	values, err = context.getReleaseValues("foo", defaultValues, defaultFiles)
	assert.NoError(t, err)
	assert.Equal(t, "c", values["a"])
	assert.Equal(t, 3, types.NewValue(values["b"].(map[string]any)["c"]).Int())
	assert.Equal(t, "foo", values["d"].(map[string]any)["e"])
	assert.Equal(t, "baz", values["d"].(map[string]any)["f"])
}
