// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package codegen

import "path"

// ResourceReaderOptions contains options for generating a resource reader
type ResourceReaderOptions struct {
	Location Location
	Package  Package
	Types    ResourceReaderTypes
}

// ResourceReaderTypes contains types for generating a resource reader
type ResourceReaderTypes struct {
	Interface string
	Struct    string
}

func generateResourceReader(options ResourceOptions) error {
	return generateTemplate(getTemplate("resourcereader.tpl"), path.Join(options.Reader.Location.Path, options.Reader.Location.File), options)
}
