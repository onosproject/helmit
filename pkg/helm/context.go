// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package helm

import (
	"encoding/csv"
	"fmt"
	"gopkg.in/yaml.v3"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/cli/values"
	"helm.sh/helm/v3/pkg/getter"
	"path/filepath"
	"reflect"
	"strings"
)

// Context is a Helm context
type Context struct {
	// Namespace is the Helm namespace
	Namespace string

	// WorkDir is the Helm working directory
	WorkDir string

	// Values is a mapping of release values
	Values map[string][]string

	// ValueFiles is a mapping of release value files
	ValueFiles map[string][]string
}

func (c *Context) getReleaseValues(release string, defaultValues map[string]any, defaultFiles []string) (map[string]any, error) {
	var valueFiles []string
	for _, valueFile := range append(defaultFiles, c.ValueFiles[release]...) {
		absPath, err := filepath.Abs(valueFile)
		if err != nil {
			return nil, err
		}
		valueFiles = append(valueFiles, absPath)
	}

	opts := &values.Options{
		ValueFiles: valueFiles,
		Values:     c.Values[release],
	}
	overrides, err := opts.MergeValues(getter.All(settings))
	if err != nil {
		return nil, err
	}
	return mergeValues(defaultValues, overrides)
}

func mergeValues(a, b map[string]any) (map[string]any, error) {
	m, err := normalizeMap(a)
	if err != nil {
		return nil, err
	}
	return mergeMaps(m, b), nil
}

func mergeMaps(a, b map[string]any) map[string]any {
	out := make(map[string]any, len(a))
	for k, v := range a {
		out[k] = v
	}
	for k, v := range b {
		if v, ok := v.(map[string]any); ok {
			if bv, ok := out[k]; ok {
				if bv, ok := bv.(map[string]any); ok {
					out[k] = mergeMaps(bv, v)
					continue
				}
			}
		}
		out[k] = v
	}
	return out
}

// getValue gets the value for the given path
func getValue(config map[string]any, path []string) any {
	names, key := getPathAndKey(path)
	parent := getMap(config, names)
	return parent[key]
}

// getMap gets the map at the given path
func getMap(parent map[string]any, path []string) map[string]any {
	if len(path) == 0 {
		return parent
	}
	child, ok := parent[path[0]]
	if !ok {
		return make(map[string]any)
	}
	return getMap(child.(map[string]any), path[1:])
}

// setKey sets a key in a map
func setKey(config map[string]any, path []string, value any) {
	names, key := getPathAndKey(path)
	parent := getMapRef(config, names)
	parent[key] = value
}

// getMapRef gets the given map reference
func getMapRef(parent map[string]any, path []string) map[string]any {
	if len(path) == 0 {
		return parent
	}
	child, ok := parent[path[0]]
	if !ok {
		child = make(map[string]any)
		parent[path[0]] = child
	}
	return getMapRef(child.(map[string]any), path[1:])
}

func getPathNames(path string) []string {
	r := csv.NewReader(strings.NewReader(path))
	r.Comma = '.'
	names, err := r.Read()
	if err != nil {
		panic(err)
	}
	return names
}

func getPathAndKey(path []string) ([]string, string) {
	return path[:len(path)-1], path[len(path)-1]
}

func normalize(value any) (any, error) {
	if value == nil {
		return nil, nil
	}
	t := reflect.TypeOf(value)
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	switch t.Kind() {
	case reflect.Struct:
		bytes, err := yaml.Marshal(value)
		if err != nil {
			return nil, err
		}
		value = make(map[string]any)
		if err := yaml.Unmarshal(bytes, &value); err != nil {
			return nil, err
		}
		return value, nil
	case reflect.Map:
		return normalizeMap(value.(map[string]any))
	case reflect.Slice:
		return normalizeSlice(value.([]any))
	}
	return value, nil
}

func normalizeMap(values map[string]any) (map[string]any, error) {
	normalized := make(map[string]any)
	if values == nil {
		return normalized, nil
	}
	for key, value := range values {
		if value != nil {
			v, err := normalize(value)
			if err != nil {
				return nil, err
			}
			normalized[key] = v
		}
	}
	return normalized, nil
}

func normalizeSlice(values []any) ([]any, error) {
	normalized := make([]any, 0, len(values))
	for _, value := range values {
		if value != nil {
			v, err := normalize(value)
			if err != nil {
				return nil, err
			}
			normalized = append(normalized, v)
		}
	}
	return normalized, nil
}

func isChartInstallable(ch *chart.Chart) (bool, error) {
	switch ch.Metadata.Type {
	case "", "application":
		return true, nil
	}
	return false, fmt.Errorf("%s charts are not installable", ch.Metadata.Type)
}

func isChartUpgradable(ch *chart.Chart) (bool, error) {
	switch ch.Metadata.Type {
	case "", "application":
		return true, nil
	}
	return false, fmt.Errorf("%s charts are not upgradable", ch.Metadata.Type)
}
