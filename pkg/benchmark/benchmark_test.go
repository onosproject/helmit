// SPDX-FileCopyrightText: 2023-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package benchmark

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetSuiteName(t *testing.T) {
	assert.Equal(t, "benchmarkSuite", getSuiteName(new(benchmarkSuite)))
	assert.Equal(t, "benchmarkSuite", getSuiteName(&benchmarkSuite{}))
}

type benchmarkSuite struct {
	Suite
}
