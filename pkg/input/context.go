// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package input

// NewContext returns a new test context
func NewContext(name string, args map[string]string) *Context {
	return &Context{
		Name: name,
		args: args,
	}
}

// Context provides the test context
type Context struct {
	Name string
	args map[string]string
}
