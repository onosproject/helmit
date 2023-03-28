// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package helm

import (
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/cli"
	"log"
	"os"
	"path/filepath"
	"sync"
)

var settings = cli.New()

var namespaces = make(map[string]*action.Configuration)
var namespacesMu = &sync.Mutex{}

// NewClient creates a new Helm client from the given Context
func NewClient(context Context) *Helm {
	if err := setContextDir(context); err != nil {
		panic(err)
	}
	return &Helm{
		context: context,
	}
}

// Helm is a Helm client
type Helm struct {
	context Context
}

// Namespace returns the Helm namespace
func (helm *Helm) Namespace() string {
	return helm.context.Namespace
}

// Repo creates a new repo command
func (helm *Helm) Repo() *RepoCmd {
	return newRepoCmd(helm.context)
}

// Install creates a new command for installing a Helm chart
func (helm *Helm) Install(release string, chart string) *InstallCmd {
	return newInstallCmd(helm.context, release, chart)
}

// Upgrade creates a new command for upgrading a Helm chart release
func (helm *Helm) Upgrade(release string, chart string) *UpgradeCmd {
	return newUpgradeCmd(helm.context, release, chart)
}

// Uninstall creates a new command for uninstalling a Helm chart release
func (helm *Helm) Uninstall(release string) *UninstallCmd {
	return newUninstall(helm.context, release)
}

// getConfig gets the Helm configuration for the given namespace
func getConfig(namespace string) (*action.Configuration, error) {
	namespacesMu.Lock()
	defer namespacesMu.Unlock()
	if config, ok := namespaces[namespace]; ok {
		return config, nil
	}
	config := &action.Configuration{}
	if err := config.Init(settings.RESTClientGetter(), namespace, "configmap", log.Printf); err != nil {
		return nil, err
	}
	namespaces[namespace] = config
	return config, nil
}

func setContextDir(context Context) error {
	dir := context.WorkDir
	if dir != "" {
		if absDir, err := filepath.Abs(dir); err != nil {
			return err
		} else if err := os.Chdir(absDir); err != nil {
			return err
		}
	}
	return nil
}
