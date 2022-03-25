// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package v1

import "fmt"

// Hostname returns the service hostname
func (s *Service) Hostname(qualified bool) string {
	if !qualified {
		return s.Name
	}
	return fmt.Sprintf("%s.%s.svc.cluster.local", s.Name, s.Namespace)
}
