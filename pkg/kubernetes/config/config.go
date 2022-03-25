// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"github.com/onosproject/helmit/pkg/util/random"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"os"
)

// NamespaceEnv is the environment variable for setting the k8s namespace
const NamespaceEnv = "POD_NAMESPACE"

// GetNamespaceFromEnv gets the Kubernetes namespace from the environment
func GetNamespaceFromEnv() string {
	namespace := os.Getenv(NamespaceEnv)
	if namespace == "" {
		namespace = random.NewPetName(2)
	}
	return namespace
}

// GetRestConfigOrDie returns the Kubernetes REST API configuration
func GetRestConfigOrDie() *rest.Config {
	config, err := GetRestConfig()
	if err != nil {
		panic(err)
	}
	return config
}

// GetRestConfig returns the Kubernetes REST API configuration
func GetRestConfig() (*rest.Config, error) {
	restconfig, err := rest.InClusterConfig()
	if err == nil {
		return restconfig, nil
	}

	kubeconfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	)
	return kubeconfig.ClientConfig()
}
