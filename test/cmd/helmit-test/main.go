// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"github.com/onosproject/helmit/pkg/test"
	tests "github.com/onosproject/helmit/test"
)

func main() {
	test.Main(map[string]test.TestingSuite{
		"chart": new(tests.ChartTestSuite),
	})
}
