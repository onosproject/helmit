// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package codegen

import "path"

// ResourceClientOptions contains options for generating a resource client
type ResourceClientOptions struct {
	Location Location
	Package  Package
	Types    ResourceClientTypes
}

// ResourceClientTypes contains types for generating a resource client
type ResourceClientTypes struct {
	Interface string
	Struct    string
}

func generateResourceClient(options ResourceOptions) error {
	return generateTemplate(getTemplate("resourceclient.tpl"), path.Join(options.Client.Location.Path, options.Client.Location.File), options)
}
