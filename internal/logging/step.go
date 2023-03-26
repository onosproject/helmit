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
	writer       = os.Stdout
	runningColor = color.New(color.FgBlue)
	successColor = color.New(color.FgGreen)
	failureColor = color.New(color.FgRed, color.Bold)
	errorColor   = color.New(color.FgRed)
)

const (
	startIcon   = "‣"
	successIcon = "✓"
	failureIcon = "✗"
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
func NewStep(job, name string, args ...interface{}) *Step {
	return &Step{
		job:     job,
		message: fmt.Sprintf(name, args...),
		verbose: GetVerbose(),
	}
}

// Step is a loggable step
type Step struct {
	job     string
	message string
	verbose bool
}

// Log logs a progress message
func (s *Step) Log(message string) {
	if s.verbose {
		fmt.Fprintf(writer, "  %s %s %s\n", time.Now().Format(time.RFC3339), s.job, message)
	}
}

// Logf logs a progress message
func (s *Step) Logf(message string, args ...interface{}) {
	if s.verbose {
		fmt.Fprintf(writer, "  %s %s %s\n", time.Now().Format(time.RFC3339), s.job, fmt.Sprintf(message, args...))
	}
}

// Start starts the step
func (s *Step) Start() {
	runningColor.Fprintf(writer, "%s %s %s %s...\n", startIcon, time.Now().Format(time.RFC3339), s.job, s.message)
}

// Complete completes the step
func (s *Step) Complete() {
	successColor.Fprintf(writer, "%s %s %s %s\n", successIcon, time.Now().Format(time.RFC3339), s.job, s.message)
}

// Fail fails the step with the given error
func (s *Step) Fail(err error) {
	failureColor.Fprintf(writer, "%s %s %s %s\n", failureIcon, time.Now().Format(time.RFC3339), s.job, s.message)
	errorColor.Fprintf(writer, "  %s\n", err.Error())
}
