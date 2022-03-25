// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package input

// NewValue returns a new Value for the given value
func NewValue(value interface{}) Value {
	return interfaceValue{
		value: value,
	}
}

// Value is a parameter value
type Value interface {
	// Interface returns the value as an interface{}
	Interface() interface{}

	// Bytes returns the value as a []byte
	Bytes() []byte

	// String returns the value as a string
	String() string

	// Slice returns the value as a slice of values
	Slice() []Value

	// Int returns the value as an int
	Int() int

	// Int32 returns the value as an int32
	Int32() int32

	// Int64 returns the value as an int64
	Int64() int64

	// Uint returns the value as a uint
	Uint() uint

	// Uint32 returns the value as a uint32
	Uint32() uint32

	// Uint64 returns the value as a uint64
	Uint64() uint64

	// Float32 returns the value as a float32
	Float32() float32

	// Float64 returns the value as a float64
	Float64() float64
}

// interfaceValue is the default implementation of the Value interface
// This implementation casts interface{} values to types based on the method call.
type interfaceValue struct {
	value interface{}
}

func (v interfaceValue) Interface() interface{} {
	return v.value
}

func (v interfaceValue) Bytes() []byte {
	return v.value.([]byte)
}

func (v interfaceValue) String() string {
	return v.value.(string)
}

func (v interfaceValue) Slice() []Value {
	return v.value.([]Value)
}

func (v interfaceValue) Int() int {
	return v.value.(int)
}

func (v interfaceValue) Int32() int32 {
	return v.value.(int32)
}

func (v interfaceValue) Int64() int64 {
	return v.value.(int64)
}

func (v interfaceValue) Uint() uint {
	return v.value.(uint)
}

func (v interfaceValue) Uint32() uint32 {
	return v.value.(uint32)
}

func (v interfaceValue) Uint64() uint64 {
	return v.value.(uint64)
}

func (v interfaceValue) Float32() float32 {
	return v.value.(float32)
}

func (v interfaceValue) Float64() float64 {
	return v.value.(float64)
}
