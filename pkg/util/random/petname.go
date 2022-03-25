// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package random

import "github.com/dustinkirkland/golang-petname"

// NewPetName returns a new random pet name
func NewPetName(words int) string {
	return petname.Generate(words, "-")
}
