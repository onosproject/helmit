// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package codegen

import "path"

// ResourceOptions contains options for generating a resource
type ResourceOptions struct {
	Client    *ResourceClientOptions
	Reader    *ResourceReaderOptions
	Reference *ResourceReferenceOptions
	Resource  *ResourceObjectOptions
	Group     *GroupOptions
}

// ResourceObjectOptions contains options for generating a resource object
type ResourceObjectOptions struct {
	Location   Location
	Package    Package
	Client     ResourceClientKind
	Kind       ResourceObjectKind
	Types      ResourceObjectTypes
	Names      ResourceObjectNames
	References []*ResourceOptions
}

// ResourceObjectKind contains kinds for generating a resource kind
type ResourceObjectKind struct {
	Package  Package
	Group    string
	Version  string
	Kind     string
	ListKind string
	Scoped   bool
}

// ResourceClientKind contains information about a resource client
type ResourceClientKind struct {
	Package Package
}

// ResourceObjectTypes contains types for generating a resource object
type ResourceObjectTypes struct {
	Kind     string
	Resource string
	Struct   string
}

// ResourceObjectNames contains names for generating a resource object
type ResourceObjectNames struct {
	Singular string
	Plural   string
}

func generateResource(options ResourceOptions) error {
	return generateTemplate(getTemplate("resource.tpl"), path.Join(options.Resource.Location.Path, options.Resource.Location.File), options)
}
