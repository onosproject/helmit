// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package codegen

import (
	"fmt"
	"path"
	"strings"
)

// Location is the location of a code file
type Location struct {
	Path string
	File string
}

// Package is the package for a code file
type Package struct {
	Name  string
	Path  string
	Alias string
}

func getOptionsFromConfig(config Config) ClientOptions {
	options := ClientOptions{
		Location: Location{
			Path: config.Path,
			File: "client.go",
		},
		Package: Package{
			Name:  path.Base(config.Package),
			Path:  config.Package,
			Alias: path.Base(config.Package),
		},
		Types: ClientTypes{
			Interface: "Client",
			Struct:    "client",
		},
		Groups: make(map[string]*GroupOptions),
	}

	for _, resource := range config.Resources {
		group := resource.Group
		if group == "" {
			group = "core"
		}
		index := strings.Index(group, ".")
		if index != -1 {
			group = group[:index]
		}

		versionOpts, ok := options.Groups[fmt.Sprintf("%s%s", resource.Group, resource.Version)]
		if !ok {
			versionOpts = &GroupOptions{
				Location: Location{
					Path: fmt.Sprintf("%s/%s/%s", config.Path, group, resource.Version),
					File: "client.go",
				},
				Package: Package{
					Name:  resource.Version,
					Path:  fmt.Sprintf("%s/%s/%s", config.Package, group, resource.Version),
					Alias: fmt.Sprintf("%s%s", group, resource.Version),
				},
				Group:   resource.Group,
				Version: resource.Version,
				Types: GroupTypes{
					Interface: "Client",
					Struct:    "client",
				},
				Names: GroupNames{
					Proper: fmt.Sprintf("%s%s", upperFirst(group), upperFirst(resource.Version)),
				},
				Resources: make(map[string]*ResourceOptions),
			}
			options.Groups[fmt.Sprintf("%s%s", resource.Group, resource.Version)] = versionOpts
		}

		_, ok = versionOpts.Resources[resource.Kind]
		if !ok {
			api := resource.API
			if api == "" {
				api = "k8s.io/api"
			}
			pkg := fmt.Sprintf("%s/%s/%s", api, group, resource.Version)

			client := resource.Client
			if client == "" {
				client = "k8s.io/client-go/kubernetes"
			}

			resourceOpts := &ResourceOptions{
				Client: &ResourceClientOptions{
					Location: Location{
						Path: fmt.Sprintf("%s/%s/%s", config.Path, group, resource.Version),
						File: fmt.Sprintf("%sclient.go", toLowerCase(resource.PluralKind)),
					},
					Package: Package{
						Name:  resource.Version,
						Path:  fmt.Sprintf("%s/%s/%s", config.Package, group, resource.Version),
						Alias: fmt.Sprintf("%s%s", group, resource.Version),
					},
					Types: ResourceClientTypes{
						Interface: fmt.Sprintf("%sClient", resource.PluralKind),
						Struct:    toLowerCamelCase(fmt.Sprintf("%sClient", resource.PluralKind)),
					},
				},
				Reader: &ResourceReaderOptions{
					Location: Location{
						Path: fmt.Sprintf("%s/%s/%s", config.Path, group, resource.Version),
						File: fmt.Sprintf("%sreader.go", toLowerCase(resource.PluralKind)),
					},
					Package: Package{
						Name:  resource.Version,
						Path:  fmt.Sprintf("%s/%s/%s", config.Package, group, resource.Version),
						Alias: fmt.Sprintf("%s%s", group, resource.Version),
					},
					Types: ResourceReaderTypes{
						Interface: fmt.Sprintf("%sReader", resource.PluralKind),
						Struct:    toLowerCamelCase(fmt.Sprintf("%sReader", resource.PluralKind)),
					},
				},
				Resource: &ResourceObjectOptions{
					Location: Location{
						Path: fmt.Sprintf("%s/%s/%s", config.Path, group, resource.Version),
						File: fmt.Sprintf("%s.go", toLowerCase(resource.Kind)),
					},
					Package: Package{
						Name:  resource.Version,
						Path:  fmt.Sprintf("%s/%s/%s", config.Package, group, resource.Version),
						Alias: fmt.Sprintf("%s%s", group, resource.Version),
					},
					Client: ResourceClientKind{
						Package: Package{
							Name:  path.Base(client),
							Path:  client,
							Alias: path.Base(client),
						},
					},
					Kind: ResourceObjectKind{
						Package: Package{
							Name:  path.Base(pkg),
							Path:  pkg,
							Alias: fmt.Sprintf("%s%s", group, resource.Version),
						},
						Group:    resource.Group,
						Version:  resource.Version,
						Kind:     resource.Kind,
						ListKind: resource.ListKind,
						Scoped:   resource.Scope != "Cluster",
					},
					Types: ResourceObjectTypes{
						Kind:     fmt.Sprintf("%sKind", resource.Kind),
						Resource: fmt.Sprintf("%sResource", resource.Kind),
						Struct:   resource.Kind,
					},
					Names: ResourceObjectNames{
						Singular: resource.Kind,
						Plural:   resource.PluralKind,
					},
				},
				Group: versionOpts,
			}
			versionOpts.Resources[resource.Kind] = resourceOpts
		}
	}

	for _, resource := range config.Resources {
		if resource.SubResources == nil {
			continue
		}
		references := make([]*ResourceOptions, 0, len(resource.SubResources))
		for _, ref := range resource.SubResources {
			groupOpts, ok := options.Groups[fmt.Sprintf("%s%s", ref.Group, ref.Version)]
			if !ok {
				continue
			}
			resourceOpts, ok := groupOpts.Resources[ref.Kind]
			if !ok {
				continue
			}
			group := resourceOpts.Resource.Kind.Group
			if group == "" {
				group = "core"
			}
			index := strings.Index(group, ".")
			if index != -1 {
				group = group[:index]
			}
			if resourceOpts.Reference == nil {
				resourceOpts.Reference = &ResourceReferenceOptions{
					Location: Location{
						Path: resourceOpts.Resource.Location.Path,
						File: fmt.Sprintf("%sreference.go", toLowerCase(resourceOpts.Resource.Names.Plural)),
					},
					Package: Package{
						Name:  resourceOpts.Resource.Kind.Version,
						Path:  fmt.Sprintf("%s/%s/%s", config.Package, group, resourceOpts.Resource.Kind.Version),
						Alias: fmt.Sprintf("%s%s", group, resourceOpts.Resource.Kind.Version),
					},
					Types: ResourceReaderTypes{
						Interface: fmt.Sprintf("%sReference", resourceOpts.Resource.Names.Plural),
						Struct:    toLowerCamelCase(fmt.Sprintf("%sReference", resourceOpts.Resource.Names.Plural)),
					},
				}
			}
			references = append(references, resourceOpts)
		}
		resourceOpts := options.Groups[fmt.Sprintf("%s%s", resource.Group, resource.Version)].Resources[resource.Kind]
		resourceOpts.Resource.References = references
	}
	return options
}
