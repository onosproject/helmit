// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package helm

import "path/filepath"

var context = &Context{}

// SetContext sets the Helm context
func SetContext(ctx *Context) error {
	ctxWorkDir := ctx.WorkDir
	if ctxWorkDir != "" {
		absDir, err := filepath.Abs(ctxWorkDir)
		if err != nil {
			return err
		}
		ctxWorkDir = absDir
	}

	ctxValueFiles := make(map[string][]string)
	for release, valueFiles := range ctx.ValueFiles {
		cleanValueFiles := make([]string, 0)
		for _, valueFile := range valueFiles {
			absPath, err := filepath.Abs(valueFile)
			if err != nil {
				return err
			}
			cleanValueFiles = append(cleanValueFiles, absPath)
		}
		ctxValueFiles[release] = cleanValueFiles
	}

	context = &Context{
		WorkDir:    ctxWorkDir,
		Values:     ctx.Values,
		ValueFiles: ctxValueFiles,
	}
	return nil
}

// Context is a Helm context
type Context struct {
	// WorkDir is the Helm working directory
	WorkDir string

	// Values is a mapping of release values
	Values map[string][]string

	// ValueFiles is a mapping of release value files
	ValueFiles map[string][]string
}

// Release returns the context for the given release
func (c *Context) Release(name string) *ReleaseContext {
	return &ReleaseContext{
		Values:     c.Values[name],
		ValueFiles: c.ValueFiles[name],
	}
}

// ReleaseContext is a Helm release context
type ReleaseContext struct {
	// ValueFiles is the release value files
	ValueFiles []string

	// Values is the release values
	Values []string
}
