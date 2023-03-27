// SPDX-FileCopyrightText: 2023-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package types

import (
	"fmt"
	"strconv"
)

// NewValue creates a new value
func NewValue(value any) Value {
	return Value{
		value: value,
	}
}

// Value is a Helm release value
type Value struct {
	value any
}

// String returns the value as a string
func (v Value) String() string {
	if v.value == nil {
		return ""
	}
	return fmt.Sprint(v.value)
}

// Bool returns the value as a boolean
func (v Value) Bool() bool {
	if v.value == nil {
		return false
	}
	b, err := strconv.ParseBool(fmt.Sprint(v.value))
	if err != nil {
		panic(err)
	}
	return b
}

// Int returns the value as an int
func (v Value) Int() int {
	return int(v.Int64())
}

// Int32 returns the value as an int32
func (v Value) Int32() int32 {
	if v.value == nil {
		return 0
	}
	i, err := strconv.ParseInt(fmt.Sprint(v.value), 10, 32)
	if err != nil {
		panic(err)
	}
	return int32(i)
}

// Int64 returns the value as an int64
func (v Value) Int64() int64 {
	if v.value == nil {
		return 0
	}
	i, err := strconv.ParseInt(fmt.Sprint(v.value), 10, 64)
	if err != nil {
		panic(err)
	}
	return i
}

// Uint returns the value as an int
func (v Value) Uint() uint {
	return uint(v.Uint64())
}

// Uint32 returns the value as an uint32
func (v Value) Uint32() uint32 {
	if v.value == nil {
		return 0
	}
	i, err := strconv.ParseUint(fmt.Sprint(v.value), 10, 32)
	if err != nil {
		panic(err)
	}
	return uint32(i)
}

// Uint64 returns the value as an uint64
func (v Value) Uint64() uint64 {
	if v.value == nil {
		return 0
	}
	i, err := strconv.ParseUint(fmt.Sprint(v.value), 10, 64)
	if err != nil {
		panic(err)
	}
	return i
}

// Float32 returns the value as a float32
func (v Value) Float32() float32 {
	if v.value == nil {
		return 0
	}
	f, err := strconv.ParseFloat(fmt.Sprint(v.value), 32)
	if err != nil {
		panic(err)
	}
	return float32(f)
}

// Float64 returns the value as a float64
func (v Value) Float64() float64 {
	if v.value == nil {
		return 0
	}
	f, err := strconv.ParseFloat(fmt.Sprint(v.value), 64)
	if err != nil {
		panic(err)
	}
	return f
}
