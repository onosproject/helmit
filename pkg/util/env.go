// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"fmt"
	"strings"
)

const keyValueSep = "="
const entrySep = ","

// SplitMap splits the given string into key-value pairs
func SplitMap(value string) map[string]string {
	values := strings.Split(value, entrySep)
	pairs := make(map[string]string)
	for _, pair := range values {
		if strings.Contains(pair, keyValueSep) {
			key := pair[:strings.Index(pair, keyValueSep)]
			value := pair[strings.Index(pair, keyValueSep)+1:]
			pairs[key] = value
		}
	}
	return pairs
}

// JoinMap joins the given map of key-value pairs into a single string
func JoinMap(pairs map[string]string) string {
	values := make([]string, 0, len(pairs))
	for key, value := range pairs {
		values = append(values, fmt.Sprintf("%s%s%s", key, keyValueSep, value))
	}
	return strings.Join(values, entrySep)
}
