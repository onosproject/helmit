// SPDX-FileCopyrightText: 2023-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package types

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestValue(t *testing.T) {
	assert.Equal(t, "foo", NewValue("foo").String())
	assert.True(t, NewValue("true").Bool())
	assert.False(t, NewValue("false").Bool())
	assert.Equal(t, 1, NewValue("1").Int())
	assert.Equal(t, int32(1), NewValue("1").Int32())
	assert.Equal(t, int64(1), NewValue("1").Int64())
	assert.Equal(t, uint(1), NewValue("1").Uint())
	assert.Equal(t, uint32(1), NewValue("1").Uint32())
	assert.Equal(t, uint64(1), NewValue("1").Uint64())
	assert.Equal(t, float32(1), NewValue("1.0").Float32())
	assert.Equal(t, float64(1), NewValue("1.0").Float64())
}
