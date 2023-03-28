// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package build

import (
	"github.com/onosproject/helmit/internal/logging"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestBuildBenchmarks(t *testing.T) {
	t.SkipNow()
	assert.NoError(t, Benchmarks(logging.NewLogger(os.Stdout)).Build("test-benchmarks", "github.com/onosproject/helmit/test/..."))
}
