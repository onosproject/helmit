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

func NewClient(context Context) *Helm {
	if err := setContextDir(context); err != nil {
		panic(err)
	}
	return &Helm{
		context: context,
	}
}

type Helm struct {
	context Context
}

func (helm *Helm) setContextDir() error {
	dir := helm.context.WorkDir
	if dir != "" {
		if absDir, err := filepath.Abs(dir); err != nil {
			return err
		} else if err := os.Chdir(absDir); err != nil {
			return err
		}
	}
	return nil
}

func (helm *Helm) Namespace() string {
	return helm.context.Namespace
}

func (helm *Helm) Repo() *RepoCmd {
	return newRepo(helm.context)
}

func (helm *Helm) Install(release string, chart string) *InstallCmd {
	return newInstall(helm.context, release, chart)
}

func (helm *Helm) Upgrade(release string, chart string) *UpgradeCmd {
	return newUpgrade(helm.context, release, chart)
}

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
