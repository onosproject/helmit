// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package test

import "github.com/onosproject/helmit/pkg/registry"

// Register registers a test suite
// Deprecated: Use registry.RegisterTestSuite instead
func Register(name string, suite TestingSuite) {
	registry.RegisterTestSuite(name, suite)
}
