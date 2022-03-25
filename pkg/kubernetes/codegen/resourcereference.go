// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package codegen

import "path"

// ResourceReferenceOptions contains options for generating a resource reference
type ResourceReferenceOptions struct {
	Location Location
	Package  Package
	Types    ResourceReaderTypes
}

// ResourceReferenceTypes contains types for generating a resource reference
type ResourceReferenceTypes struct {
	Interface string
	Struct    string
}

func generateResourceReference(options ResourceOptions) error {
	if options.Reference == nil {
		return nil
	}
	return generateTemplate(getTemplate("resourcereference.tpl"), path.Join(options.Reference.Location.Path, options.Reference.Location.File), options)
}
