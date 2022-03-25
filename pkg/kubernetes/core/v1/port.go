// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	"fmt"
	"github.com/onosproject/helmit/pkg/kubernetes/resource"
	corev1 "k8s.io/api/core/v1"
)

// ServicePort is a service port
type ServicePort struct {
	resource.Client
	corev1.ServicePort
	service *Service
}

// Address returns the address of the port
func (p *ServicePort) Address(qualified bool) string {
	return fmt.Sprintf("%s:%d", p.service.Hostname(qualified), p.Port)
}

// Ports returns a list of service ports
func (s *Service) Ports() []*ServicePort {
	ports := make([]*ServicePort, len(s.Object.Spec.Ports))
	for i, port := range s.Object.Spec.Ports {
		ports[i] = &ServicePort{
			Client:      s.Resource.Client,
			ServicePort: port,
			service:     s,
		}
	}
	return ports
}

// Port returns a service port by name
func (s *Service) Port(name string) *ServicePort {
	for _, port := range s.Object.Spec.Ports {
		if port.Name == name {
			return &ServicePort{
				Client:      s.Resource.Client,
				ServicePort: port,
				service:     s,
			}
		}
	}
	return nil
}
