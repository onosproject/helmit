// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package logging

import (
	"fmt"
	"io"
	"time"
)

// NewLogger creates a new logger to the given io.Writer
func NewLogger(writer io.Writer) Logger {
	return &logger{
		writer: writer,
	}
}

// Logger is an interface for logging messages
type Logger interface {
	// Log logs a message to the console
	Log(message string)
	// Logf logs a formatted message to the console
	Logf(message string, args ...any)
}

type logger struct {
	writer io.Writer
}

// Log logs a progress message
func (l *logger) Log(message string) {
	fmt.Fprintf(writer, "  %s %s\n", time.Now().Format(time.RFC3339), message)
}

// Logf logs a progress message
func (l *logger) Logf(message string, args ...interface{}) {
	fmt.Fprintf(writer, "  %s %s\n", time.Now().Format(time.RFC3339), fmt.Sprintf(message, args...))
}
