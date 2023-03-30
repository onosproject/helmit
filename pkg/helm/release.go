// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package helm

import (
	"context"
	"github.com/onosproject/helmit/pkg/types"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/downloader"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/release"
	"os"
	"time"
)

const defaultTimeout = 10 * time.Minute

func newReleaseCmd[T any](cmd T, context Context, release string, chart string) *ReleaseCmd[T] {
	return &ReleaseCmd[T]{
		context:   context,
		namespace: context.Namespace,
		release:   release,
		chart:     chart,
		values:    make(map[string]any),
		timeout:   defaultTimeout,
		cmd:       cmd,
	}
}

// ReleaseCmd is a base command for install/upgrade commands
type ReleaseCmd[T any] struct {
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
	cmd        T
}

// Namespace sets the namespace to which to install the chart
func (cmd *ReleaseCmd[T]) Namespace(namespace string) T {
	cmd.namespace = namespace
	return cmd.cmd
}

// Version sets the version of the chart to install
func (cmd *ReleaseCmd[T]) Version(version string) T {
	cmd.version = version
	return cmd.cmd
}

// Username sets the chart repo username
func (cmd *ReleaseCmd[T]) Username(username string) T {
	cmd.username = username
	return cmd.cmd
}

// Password sets the password for the chart repo
func (cmd *ReleaseCmd[T]) Password(password string) T {
	cmd.password = password
	return cmd.cmd
}

// RepoURL sets the URL of the repository from which to install the Helm chart
func (cmd *ReleaseCmd[T]) RepoURL(repoURL string) T {
	cmd.repoURL = repoURL
	return cmd.cmd
}

// SkipCRDs skips installing CRDs in the chart
func (cmd *ReleaseCmd[T]) SkipCRDs() T {
	cmd.skipCRDs = true
	return cmd.cmd
}

// Atomic enables atomic install
func (cmd *ReleaseCmd[T]) Atomic() T {
	cmd.atomic = true
	return cmd.cmd
}

// Verify configures the command for verifying the installation
func (cmd *ReleaseCmd[T]) Verify() T {
	cmd.verify = true
	return cmd.cmd
}

// DryRun sets the command to dry run mode
func (cmd *ReleaseCmd[T]) DryRun() T {
	cmd.dryRun = true
	return cmd.cmd
}

// Wait configures the command to wait for all resources to be running before returning Get or Do calls
func (cmd *ReleaseCmd[T]) Wait() T {
	cmd.wait = true
	return cmd.cmd
}

// Timeout sets the installation timeout
func (cmd *ReleaseCmd[T]) Timeout(timeout time.Duration) T {
	cmd.timeout = timeout
	return cmd.cmd
}

// Set sets a Helm chart value override
func (cmd *ReleaseCmd[T]) Set(path string, value interface{}) T {
	setKey(cmd.values, getPathNames(path), value)
	return cmd.cmd
}

// Values adds values files to the release
func (cmd *ReleaseCmd[T]) Values(files ...string) T {
	cmd.valueFiles = append(cmd.valueFiles, files...)
	return cmd.cmd
}

func (cmd *ReleaseCmd[T]) loadChart(options action.ChartPathOptions) (*chart.Chart, error) {
	// Locate the chart path
	path, err := options.LocateChart(cmd.chart, settings)
	if err != nil {
		return nil, err
	}

	// Check chart dependencies to make sure all are present in /charts
	chart, err := loader.Load(path)
	if err != nil {
		return nil, err
	}

	if req := chart.Metadata.Dependencies; req != nil {
		// If CheckDependencies returns an error, we have unfulfilled dependencies.
		// As of Helm 2.4.0, this is treated as a stopping condition:
		// https://github.com/helm/helm/issues/2209
		if err := action.CheckDependencies(chart, req); err != nil {
			man := &downloader.Manager{
				Out:              os.Stdout,
				ChartPath:        path,
				Keyring:          options.Keyring,
				SkipUpdate:       false,
				Getters:          getter.All(cli.New()),
				RepositoryConfig: settings.RepositoryConfig,
				RepositoryCache:  settings.RepositoryCache,
			}
			if err := man.Update(); err != nil {
				return nil, err
			}
		}
	}
	return chart, nil
}

func newInstallCmd(context Context, release string, chart string) *InstallCmd {
	cmd := &InstallCmd{}
	cmd.ReleaseCmd = newReleaseCmd[*InstallCmd](cmd, context, release, chart)
	return cmd
}

// InstallCmd is a command for installing a Helm chart
type InstallCmd struct {
	*ReleaseCmd[*InstallCmd]
}

// Do runs the command
func (cmd *InstallCmd) Do(ctx context.Context) error {
	_, err := cmd.run(ctx)
	return err
}

// Get runs the command and returns the resulting Release
func (cmd *InstallCmd) Get(ctx context.Context) (*Release, error) {
	release, err := cmd.run(ctx)
	if err != nil {
		return nil, err
	}
	values, err := mergeValues(release.Chart.Values, release.Config)
	if err != nil {
		return nil, err
	}
	return &Release{
		Namespace: release.Namespace,
		Name:      release.Name,
		values:    values,
	}, nil
}

// run runs the command
func (cmd *InstallCmd) run(ctx context.Context) (*release.Release, error) {
	config, err := getConfig(cmd.namespace)
	if err != nil {
		return nil, err
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

	chart, err := cmd.loadChart(install.ChartPathOptions)
	if err != nil {
		return nil, err
	}

	valid, err := isChartInstallable(chart)
	if !valid {
		return nil, err
	}

	values, err := cmd.context.getReleaseValues(cmd.release, cmd.values, cmd.valueFiles)
	if err != nil {
		return nil, err
	}
	return install.RunWithContext(ctx, chart, values)
}

func newUpgradeCmd(context Context, release string, chart string) *UpgradeCmd {
	cmd := &UpgradeCmd{}
	cmd.ReleaseCmd = newReleaseCmd[*UpgradeCmd](cmd, context, release, chart)
	return cmd
}

// UpgradeCmd is a command for upgrading a Helm chart
type UpgradeCmd struct {
	*ReleaseCmd[*UpgradeCmd]
	install bool
}

// Install sets the upgrade command to install mode
func (cmd *UpgradeCmd) Install() *UpgradeCmd {
	cmd.install = true
	return cmd
}

// Do runs the command
func (cmd *UpgradeCmd) Do(ctx context.Context) error {
	_, err := cmd.run(ctx)
	return err
}

// Get runs the command and returns the resulting Release
func (cmd *UpgradeCmd) Get(ctx context.Context) (*Release, error) {
	release, err := cmd.run(ctx)
	if err != nil {
		return nil, err
	}
	values, err := mergeValues(release.Chart.Values, release.Config)
	if err != nil {
		return nil, err
	}
	return &Release{
		Namespace: release.Namespace,
		Name:      release.Name,
		values:    values,
	}, nil
}

// run runs the command
func (cmd *UpgradeCmd) run(ctx context.Context) (*release.Release, error) {
	config, err := getConfig(cmd.namespace)
	if err != nil {
		return nil, err
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

	chart, err := cmd.loadChart(upgrade.ChartPathOptions)
	if err != nil {
		return nil, err
	}

	valid, err := isChartUpgradable(chart)
	if !valid {
		return nil, err
	}

	values, err := cmd.context.getReleaseValues(cmd.release, cmd.values, cmd.valueFiles)
	if err != nil {
		return nil, err
	}
	return upgrade.RunWithContext(ctx, cmd.release, chart, values)
}

func newUninstall(context Context, release string) *UninstallCmd {
	return &UninstallCmd{
		context:   context,
		namespace: context.Namespace,
		release:   release,
		timeout:   defaultTimeout,
	}
}

// UninstallCmd is a command for uninstalling a Helm chart release
type UninstallCmd struct {
	context   Context
	namespace string
	release   string
	wait      bool
	timeout   time.Duration
}

// Namespace sets the namespace in which to run the command
func (cmd *UninstallCmd) Namespace(namespace string) *UninstallCmd {
	cmd.namespace = namespace
	return cmd
}

// Wait sets the command to block until complete
func (cmd *UninstallCmd) Wait() *UninstallCmd {
	cmd.wait = true
	return cmd
}

// Timeout sets the command timeout
func (cmd *UninstallCmd) Timeout(timeout time.Duration) *UninstallCmd {
	cmd.timeout = timeout
	return cmd
}

// Do runs the command
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

// Release is a release configuration
type Release struct {
	Namespace string
	Name      string
	values    map[string]any
}

// Get gets a value from the release
func (r *Release) Get(path string) Value {
	return Value{
		Path:  path,
		Value: types.NewValue(getValue(r.values, getPathNames(path))),
	}
}

// Value is a Helm release value
type Value struct {
	Path string
	types.Value
}
