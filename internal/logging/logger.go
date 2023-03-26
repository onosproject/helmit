// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package logging

// Logger is an interface for logging messages
type Logger interface {
	// Log logs a message to the console
	Log(message string)
	// Logf logs a formatted message to the console
	Logf(message string, args ...any)
}
