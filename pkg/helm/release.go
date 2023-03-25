package helm

import (
	"context"
	"fmt"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/downloader"
	"helm.sh/helm/v3/pkg/getter"
	"os"
	"strconv"
	"time"
)

const defaultTimeout = 10 * time.Minute

func newInstall(context Context, release string, chart string) *InstallCmd {
	return &InstallCmd{
		context:   context,
		namespace: context.Namespace,
		release:   release,
		chart:     chart,
		values:    make(map[string]any),
		timeout:   defaultTimeout,
	}
}

type InstallCmd struct {
	context    Context
	namespace  string
	release    string
	chart      string
	version    string
	repoURL    string
	username   string
	password   string
	skipCRDs   bool
	atomic     bool
	verify     bool
	dryRun     bool
	wait       bool
	timeout    time.Duration
	values     map[string]any
	valueFiles []string
}

func (cmd *InstallCmd) Namespace(namespace string) *InstallCmd {
	cmd.namespace = namespace
	return cmd
}

func (cmd *InstallCmd) Version(version string) *InstallCmd {
	cmd.version = version
	return cmd
}

func (cmd *InstallCmd) Username(username string) *InstallCmd {
	cmd.username = username
	return cmd
}

func (cmd *InstallCmd) Password(password string) *InstallCmd {
	cmd.password = password
	return cmd
}

func (cmd *InstallCmd) RepoURL(repoURL string) *InstallCmd {
	cmd.repoURL = repoURL
	return cmd
}

func (cmd *InstallCmd) SkipCRDs() *InstallCmd {
	cmd.skipCRDs = true
	return cmd
}

func (cmd *InstallCmd) Atomic() *InstallCmd {
	cmd.atomic = true
	return cmd
}

func (cmd *InstallCmd) Verify() *InstallCmd {
	cmd.verify = true
	return cmd
}

func (cmd *InstallCmd) DryRun() *InstallCmd {
	cmd.dryRun = true
	return cmd
}

func (cmd *InstallCmd) Wait() *InstallCmd {
	cmd.wait = true
	return cmd
}

func (cmd *InstallCmd) Timeout(timeout time.Duration) *InstallCmd {
	cmd.timeout = timeout
	return cmd
}

// Set sets a value
func (cmd *InstallCmd) Set(path string, value interface{}) *InstallCmd {
	setKey(cmd.values, getPathNames(path), value)
	return cmd
}

// Values adds values files to the release
func (cmd *InstallCmd) Values(files ...string) *InstallCmd {
	cmd.valueFiles = append(cmd.valueFiles, files...)
	return cmd
}

func (cmd *InstallCmd) Do(ctx context.Context) error {
	values, err := cmd.context.getReleaseValues(cmd.release, cmd.values, cmd.valueFiles)
	if err != nil {
		return err
	}
	return cmd.do(ctx, values)
}

func (cmd *InstallCmd) Get(ctx context.Context) (*Release, error) {
	values, err := cmd.context.getReleaseValues(cmd.release, cmd.values, cmd.valueFiles)
	if err != nil {
		return nil, err
	}
	if err := cmd.do(ctx, values); err != nil {
		return nil, err
	}
	return &Release{
		Namespace: cmd.namespace,
		Name:      cmd.release,
		values:    values,
	}, nil
}

// do installs the Helm chart
func (cmd *InstallCmd) do(ctx context.Context, values map[string]any) error {
	config, err := getConfig(cmd.namespace)
	if err != nil {
		return err
	}

	install := action.NewInstall(config)
	install.Namespace = cmd.namespace
	install.Version = cmd.version
	install.Username = cmd.username
	install.Password = cmd.password
	install.SkipCRDs = cmd.skipCRDs
	install.RepoURL = cmd.repoURL
	install.ReleaseName = cmd.release
	install.Atomic = cmd.atomic
	install.Wait = cmd.wait
	install.Verify = cmd.verify
	install.DryRun = cmd.dryRun
	install.Timeout = cmd.timeout

	// Locate the chart path
	path, err := install.ChartPathOptions.LocateChart(cmd.chart, settings)
	if err != nil {
		return err
	}

	// Check chart dependencies to make sure all are present in /charts
	chart, err := loader.Load(path)
	if err != nil {
		return err
	}

	valid, err := isChartInstallable(chart)
	if !valid {
		return err
	}

	if req := chart.Metadata.Dependencies; req != nil {
		// If CheckDependencies returns an error, we have unfulfilled dependencies.
		// As of Helm 2.4.0, this is treated as a stopping condition:
		// https://github.com/helm/helm/issues/2209
		if err := action.CheckDependencies(chart, req); err != nil {
			if install.DependencyUpdate {
				man := &downloader.Manager{
					Out:              os.Stdout,
					ChartPath:        path,
					Keyring:          install.ChartPathOptions.Keyring,
					SkipUpdate:       false,
					Getters:          getter.All(cli.New()),
					RepositoryConfig: settings.RepositoryConfig,
					RepositoryCache:  settings.RepositoryCache,
				}
				if err := man.Update(); err != nil {
					return err
				}
			} else {
				return err
			}
		}
	}

	if _, err := install.RunWithContext(ctx, chart, values); err != nil {
		return err
	}
	return nil
}

func newUpgrade(context Context, release string, chart string) *UpgradeCmd {
	return &UpgradeCmd{
		context:   context,
		namespace: context.Namespace,
		release:   release,
		chart:     chart,
		values:    make(map[string]any),
		timeout:   defaultTimeout,
	}
}

type UpgradeCmd struct {
	context    Context
	namespace  string
	release    string
	chart      string
	version    string
	install    bool
	repoURL    string
	username   string
	password   string
	skipCRDs   bool
	atomic     bool
	wait       bool
	timeout    time.Duration
	verify     bool
	dryRun     bool
	values     map[string]any
	valueFiles []string
}

func (cmd *UpgradeCmd) Namespace(namespace string) *UpgradeCmd {
	cmd.namespace = namespace
	return cmd
}

func (cmd *UpgradeCmd) Install() *UpgradeCmd {
	cmd.install = true
	return cmd
}

func (cmd *UpgradeCmd) Version(version string) *UpgradeCmd {
	cmd.version = version
	return cmd
}

func (cmd *UpgradeCmd) Username(username string) *UpgradeCmd {
	cmd.username = username
	return cmd
}

func (cmd *UpgradeCmd) Password(password string) *UpgradeCmd {
	cmd.password = password
	return cmd
}

func (cmd *UpgradeCmd) RepoURL(repoURL string) *UpgradeCmd {
	cmd.repoURL = repoURL
	return cmd
}

func (cmd *UpgradeCmd) SkipCRDs() *UpgradeCmd {
	cmd.skipCRDs = true
	return cmd
}

func (cmd *UpgradeCmd) Atomic() *UpgradeCmd {
	cmd.atomic = true
	return cmd
}

func (cmd *UpgradeCmd) Verify() *UpgradeCmd {
	cmd.verify = true
	return cmd
}

func (cmd *UpgradeCmd) DryRun() *UpgradeCmd {
	cmd.dryRun = true
	return cmd
}

func (cmd *UpgradeCmd) Wait() *UpgradeCmd {
	cmd.wait = true
	return cmd
}

func (cmd *UpgradeCmd) Timeout(timeout time.Duration) *UpgradeCmd {
	cmd.timeout = timeout
	return cmd
}

// Set sets a value
func (cmd *UpgradeCmd) Set(path string, value interface{}) *UpgradeCmd {
	setKey(cmd.values, getPathNames(path), value)
	return cmd
}

// Values adds values files to the release
func (cmd *UpgradeCmd) Values(files ...string) *UpgradeCmd {
	cmd.valueFiles = append(cmd.valueFiles, files...)
	return cmd
}

func (cmd *UpgradeCmd) Do(ctx context.Context) error {
	values, err := cmd.context.getReleaseValues(cmd.release, cmd.values, cmd.valueFiles)
	if err != nil {
		return err
	}
	return cmd.do(ctx, values)
}

func (cmd *UpgradeCmd) Get(ctx context.Context) (*Release, error) {
	values, err := cmd.context.getReleaseValues(cmd.release, cmd.values, cmd.valueFiles)
	if err != nil {
		return nil, err
	}
	if err := cmd.do(ctx, values); err != nil {
		return nil, err
	}
	return &Release{
		Namespace: cmd.namespace,
		Name:      cmd.release,
		values:    values,
	}, nil
}

// do installs the Helm chart
func (cmd *UpgradeCmd) do(ctx context.Context, values map[string]any) error {
	config, err := getConfig(cmd.namespace)
	if err != nil {
		return err
	}

	upgrade := action.NewUpgrade(config)
	upgrade.Namespace = cmd.namespace
	upgrade.Install = cmd.install
	upgrade.Version = cmd.version
	upgrade.Username = cmd.username
	upgrade.Password = cmd.password
	upgrade.SkipCRDs = cmd.skipCRDs
	upgrade.RepoURL = cmd.repoURL
	upgrade.Atomic = cmd.atomic
	upgrade.DryRun = cmd.dryRun
	upgrade.Verify = cmd.verify
	upgrade.Wait = cmd.wait
	upgrade.Timeout = cmd.timeout

	// Locate the chart path
	path, err := upgrade.ChartPathOptions.LocateChart(cmd.chart, settings)
	if err != nil {
		return err
	}

	// Check chart dependencies to make sure all are present in /charts
	chart, err := loader.Load(path)
	if err != nil {
		return err
	}

	valid, err := isChartUpgradable(chart)
	if !valid {
		return err
	}

	if req := chart.Metadata.Dependencies; req != nil {
		// If CheckDependencies returns an error, we have unfulfilled dependencies.
		// As of Helm 2.4.0, this is treated as a stopping condition:
		// https://github.com/helm/helm/issues/2209
		if err := action.CheckDependencies(chart, req); err != nil {
			if upgrade.DependencyUpdate {
				man := &downloader.Manager{
					Out:              os.Stdout,
					ChartPath:        path,
					Keyring:          upgrade.ChartPathOptions.Keyring,
					SkipUpdate:       false,
					Getters:          getter.All(cli.New()),
					RepositoryConfig: settings.RepositoryConfig,
					RepositoryCache:  settings.RepositoryCache,
				}
				if err := man.Update(); err != nil {
					return err
				}
			} else {
				return err
			}
		}
	}

	if _, err := upgrade.RunWithContext(ctx, cmd.release, chart, values); err != nil {
		return err
	}
	return nil
}

func newUninstall(context Context, release string) *UninstallCmd {
	return &UninstallCmd{
		context:   context,
		namespace: context.Namespace,
		release:   release,
		timeout:   defaultTimeout,
	}
}

type UninstallCmd struct {
	context   Context
	namespace string
	release   string
	wait      bool
	timeout   time.Duration
}

func (cmd *UninstallCmd) Namespace(namespace string) *UninstallCmd {
	cmd.namespace = namespace
	return cmd
}

func (cmd *UninstallCmd) Wait() *UninstallCmd {
	cmd.wait = true
	return cmd
}

func (cmd *UninstallCmd) Timeout(timeout time.Duration) *UninstallCmd {
	cmd.timeout = timeout
	return cmd
}

func (cmd *UninstallCmd) Do(ctx context.Context) error {
	config, err := getConfig(cmd.namespace)
	if err != nil {
		return err
	}

	uninstall := action.NewUninstall(config)
	uninstall.Wait = cmd.wait
	uninstall.Timeout = cmd.timeout
	_, err = uninstall.Run(cmd.release)
	return err
}

type Release struct {
	Namespace string
	Name      string
	values    map[string]any
}

func (r *Release) Get(path string) Value {
	return Value{
		value: getValue(r.values, getPathNames(path)),
	}
}

type Value struct {
	value any
}

func (v Value) String() string {
	return fmt.Sprint(v.value)
}

func (v Value) Bool() bool {
	b, err := strconv.ParseBool(fmt.Sprint(v.value))
	if err != nil {
		panic(err)
	}
	return b
}

func (v Value) Int() int {
	i, err := strconv.Atoi(fmt.Sprint(v.value))
	if err != nil {
		panic(err)
	}
	return i
}

func (v Value) Int32() int32 {
	return int32(v.Int())
}

func (v Value) Int64() int64 {
	return int64(v.Int())
}
