package helm

import (
	"context"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/cli"
	"log"
)

var settings = cli.New()

type Cmd[T any] interface {
	Do(ctx context.Context) error
}

func NewClient(context Context) *Helm {
	return &Helm{
		context: context,
	}
}

type Helm struct {
	context Context
}

func (h *Helm) Namespace() string {
	return h.context.Namespace
}

func (h *Helm) Repo() *RepoCmd {
	return newRepo(h.context)
}

func (h *Helm) Install(release string, chart string) *InstallCmd {
	return newInstall(h.context, release, chart)
}

func (h *Helm) Upgrade(release string, chart string) *UpgradeCmd {
	return newUpgrade(h.context, release, chart)
}

func (h *Helm) Uninstall(release string) *UninstallCmd {
	return newUninstall(h.context, release)
}

// getConfig gets the Helm configuration for the given namespace
func getConfig(namespace string) (*action.Configuration, error) {
	config := &action.Configuration{}
	if err := config.Init(settings.RESTClientGetter(), namespace, "memory", log.Printf); err != nil {
		return nil, err
	}
	return config, nil
}
