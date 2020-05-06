// Copyright 2020-present Open Networking Foundation.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/onosproject/helmit/api/helm"
	helmapi "github.com/onosproject/helmit/pkg/helm"
	service "github.com/onosproject/onos-lib-go/pkg/northbound"
	"google.golang.org/grpc"
)

const NamespaceEnv = "POD_NAMESPACE"

// NewService returns a new helm Service
func NewService() (service.Service, error) {
	return &Service{}, nil
}

// Service is an implementation of C1 service.
type Service struct {
	service.Service
}

// Register registers the helm Service with the gRPC server.
func (s Service) Register(r *grpc.Server) {
	server := &Server{
		releases: make(map[string]*helmapi.HelmRelease),
	}
	helm.RegisterHelmServer(r, server)
}

type Server struct {
	releases map[string]*helmapi.HelmRelease
}

// Uninstall uninstall a helm chart
func (s *Server) Uninstall(ctx context.Context, uninstalReq *helm.HelmUninstallRequest) (*helm.HelmUninstallResponse, error) {
	if _, ok := s.releases[uninstalReq.Name]; ok {
		err := s.releases[uninstalReq.Name].Uninstall()
		if err != nil {
			return &helm.HelmUninstallResponse{
				Status: helm.Status_FAILED,
			}, err
		}
		return &helm.HelmUninstallResponse{
			Status: helm.Status_SUCCEED,
		}, nil
	}

	return &helm.HelmUninstallResponse{
		Status: helm.Status_FAILED,
	}, fmt.Errorf("helm chart %s has not been installed", uninstalReq.Name)

}

// Install install a helm chart
func (s *Server) Install(ctx context.Context, releaseReq *helm.HelmReleaseRequest) (*helm.HelmReleaseResponse, error) {
	if releaseReq.Name == "" || releaseReq.Chart.Name == "" || releaseReq.Chart.Repository == "" {
		return &helm.HelmReleaseResponse{
			Status: helm.Status_FAILED,
		}, errors.New("release name, chart name, and chart repository cannot be empty")
	}

	chart := helmapi.Chart(releaseReq.Chart.Name, releaseReq.Chart.Repository)
	release := chart.Release(releaseReq.Name)
	release.SetSkipCRDs(releaseReq.SkipCRDs)
	s.releases[releaseReq.Name] = release

	for key, value := range releaseReq.Values {
		release.Set(key, value)
	}
	err := release.Install(releaseReq.Wait)
	if err != nil {
		return &helm.HelmReleaseResponse{
			Status: helm.Status_FAILED,
		}, err
	}

	return &helm.HelmReleaseResponse{
		Status: helm.Status_SUCCEED,
	}, nil
}
