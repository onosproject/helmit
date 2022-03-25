// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package input

import (
	"strconv"
)

// GetArg gets an argument
func (c *Context) GetArg(name string) *Arg {
	if value, ok := c.args[name]; ok {
		return &Arg{
			value: value,
		}
	}
	return &Arg{}
}

// Arg is an argument
type Arg struct {
	value string
}

// Int returns the argument as an int
func (a *Arg) Int(def int) int {
	if a.value == "" {
		return def
	}
	i, err := strconv.Atoi(a.value)
	if err != nil {
		panic(err)
	}
	return i
}

// String returns the argument as a string
func (a *Arg) String(def string) string {
	if a.value == "" {
		return def
	}
	return a.value
}
