// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package build

import (
	"github.com/onosproject/helmit/internal/logging"
	"github.com/onosproject/helmit/pkg/test"
	"reflect"
)

const testMainTpl = `
package main

import (
	"github.com/onosproject/helmit/pkg/test"
	{{- range .Imports }}
	{{ .Alias }} "{{ .Path }}"
	{{- end }}
)

func main() {
	test.Main([]test.TestingSuite{
		{{- range .Suites }}
		new({{ .Import.Alias }}.{{ .Name }}),
		{{- end }}
	})
}
`

const defaultTestSuiteMatcher = "TestSuite$"

// Tests returns a new test binary builder
func Tests(log logging.Logger, suiteMatchers ...string) *Builder {
	if len(suiteMatchers) == 0 {
		suiteMatchers = []string{defaultTestSuiteMatcher}
	}
	return newBuilder(reflect.TypeOf(test.Suite{}), suiteMatchers, testMainTpl, log)
}
