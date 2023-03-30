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

func TestBuildTests(t *testing.T) {
	defer os.Remove("test-tests")
	assert.NoError(t, Tests(logging.NewLogger(os.Stdout)).
		Build("test-tests", "github.com/onosproject/helmit/test/..."))
}
