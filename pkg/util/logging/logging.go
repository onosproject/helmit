// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package logging

import (
	"fmt"
	"github.com/fatih/color"
	"os"
	"time"
)

var (
	start   = "‣"
	success = "✓"
	failure = "✗"
	writer  = os.Stdout
)

const verboseEnv = "VERBOSE_LOGGING"

// GetVerbose returns whether verbose logging is enabled
func GetVerbose() bool {
	verbose := os.Getenv(verboseEnv)
	return verbose != ""
}

// SetVerbose sets verbose logging
func SetVerbose(verbose bool) {
	if verbose {
		_ = os.Setenv(verboseEnv, "true")
	} else {
		_ = os.Unsetenv(verboseEnv)
	}
}

// NewStep returns a new step
func NewStep(test, name string, args ...interface{}) *Step {
	return &Step{
		test:    test,
		name:    fmt.Sprintf(name, args...),
		verbose: GetVerbose(),
	}
}

// Step is a loggable step
type Step struct {
	test    string
	name    string
	verbose bool
}

// Log logs a progress message
func (s *Step) Log(message string) {
	if s.verbose {
		fmt.Fprintf(writer, "  %s %s %s\n", time.Now().Format(time.RFC3339), s.test, message)
	}
}

// Logf logs a progress message
func (s *Step) Logf(message string, args ...interface{}) {
	if s.verbose {
		fmt.Fprintf(writer, "  %s %s %s\n", time.Now().Format(time.RFC3339), s.test, fmt.Sprintf(message, args...))
	}
}

// Start starts the step
func (s *Step) Start() {
	fmt.Fprintln(writer, color.CyanString(fmt.Sprintf("%s %s %s %s", start, time.Now().Format(time.RFC3339), s.test, s.name)))
}

// Complete completes the step
func (s *Step) Complete() {
	fmt.Fprintln(writer, color.GreenString(fmt.Sprintf("%s %s %s %s", success, time.Now().Format(time.RFC3339), s.test, s.name)))
}

// Fail fails the step with the given error
func (s *Step) Fail(err error) {
	fmt.Fprintln(writer, color.RedString(fmt.Sprintf("%s %s %s %s", failure, time.Now().Format(time.RFC3339), s.test, s.name)))
}

// Print prints the given log line
func Print(line string) {
	if line == "" {
		return
	}
	if len(line) >= len(start) && line[:len(start)] == start {
		fmt.Fprintln(writer, color.CyanString(line))
	} else if len(line) >= len(success) && line[:len(success)] == success {
		fmt.Fprintln(writer, color.GreenString(line))
	} else if len(line) >= len(failure) && line[:len(failure)] == failure {
		fmt.Fprintln(writer, color.RedString(line))
	} else {
		fmt.Fprintln(writer, line)
	}
}
