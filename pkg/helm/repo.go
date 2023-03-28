// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package helm

import (
	"context"
	"github.com/gofrs/flock"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/helmpath"
	"helm.sh/helm/v3/pkg/repo"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func newRepoCmd(context Context) *RepoCmd {
	return &RepoCmd{
		context: context,
	}
}

// RepoCmd is a Helm repository command
type RepoCmd struct {
	context Context
}

// Add creates a Helm repository add command
func (repo *RepoCmd) Add(name string, url string) *RepoAddCmd {
	return newRepoAdd(repo.context, name, url)
}

// Remove creates a Helm repository remove command
func (repo *RepoCmd) Remove(name string) *RepoRemoveCmd {
	return newRepoRemove(repo.context, name)
}

func newRepoAdd(context Context, name string, url string) *RepoAddCmd {
	return &RepoAddCmd{
		context: context,
		name:    name,
		url:     url,
	}
}

// RepoAddCmd is a Helm repository add command
type RepoAddCmd struct {
	context  Context
	name     string
	url      string
	username string
	password string
}

// Username sets the Helm repository username
func (cmd *RepoAddCmd) Username(username string) *RepoAddCmd {
	cmd.username = username
	return cmd
}

// Password sets the Helm repository password
func (cmd *RepoAddCmd) Password(password string) *RepoAddCmd {
	cmd.password = password
	return cmd
}

// Do runs the Helm repository add command
func (cmd *RepoAddCmd) Do(ctx context.Context) error {
	repoFile := settings.RepositoryConfig

	// Ensure the file directory exists as it is required for file locking
	err := os.MkdirAll(filepath.Dir(repoFile), os.ModePerm)
	if err != nil && !os.IsExist(err) {
		return err
	}

	// Acquire a file lock for process synchronization
	repoFileExt := filepath.Ext(repoFile)
	var lockPath string
	if len(repoFileExt) > 0 && len(repoFileExt) < len(repoFile) {
		lockPath = strings.TrimSuffix(repoFile, repoFileExt) + ".lock"
	} else {
		lockPath = repoFile + ".lock"
	}
	fileLock := flock.New(lockPath)
	lockCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	locked, err := fileLock.TryLockContext(lockCtx, time.Second)
	if err == nil && locked {
		defer func() {
			_ = fileLock.Unlock()
		}()
	}
	if err != nil {
		return err
	}

	b, err := os.ReadFile(repoFile)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	var f repo.File
	if err := yaml.Unmarshal(b, &f); err != nil {
		return err
	}

	entry := &repo.Entry{
		Name:     cmd.name,
		URL:      cmd.url,
		Username: cmd.username,
		Password: cmd.password,
	}

	repo, err := repo.NewChartRepository(entry, getter.All(settings))
	if err != nil {
		return err
	}

	cachePath := settings.RepositoryCache
	if cachePath != "" {
		repo.CachePath = cachePath
	}
	if _, err := repo.DownloadIndexFile(); err != nil {
		return errors.Wrapf(err, "looks like %q is not a valid chart repository or cannot be reached", cmd.url)
	}

	f.Update(entry)

	if err := f.WriteFile(repoFile, 0644); err != nil {
		return err
	}
	return err
}

func newRepoRemove(context Context, names ...string) *RepoRemoveCmd {
	return &RepoRemoveCmd{
		context: context,
		names:   names,
	}
}

// RepoRemoveCmd is a Helm repository remove command
type RepoRemoveCmd struct {
	context Context
	names   []string
}

// Do runs the Helm repository remove command
func (cmd *RepoRemoveCmd) Do(ctx context.Context) error {
	repoFile := settings.RepositoryConfig
	repoCache := settings.RepositoryCache

	repo, err := repo.LoadFile(repoFile)
	if os.IsNotExist(err) || len(repo.Repositories) == 0 {
		return errors.New("no repositories configured")
	}

	for _, name := range cmd.names {
		if !repo.Remove(name) {
			return errors.Errorf("no repo named %q found", name)
		}
		if err := repo.WriteFile(repoFile, 0644); err != nil {
			return err
		}

		if err := removeRepoCache(repoCache, name); err != nil {
			return err
		}
	}
	return err
}

func removeRepoCache(root, name string) error {
	idx := filepath.Join(root, helmpath.CacheChartsFile(name))
	if _, err := os.Stat(idx); err == nil {
		_ = os.Remove(idx)
	}

	idx = filepath.Join(root, helmpath.CacheIndexFile(name))
	if _, err := os.Stat(idx); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return errors.Wrapf(err, "can't remove index file %s", idx)
	}
	return os.Remove(idx)
}
