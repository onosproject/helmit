// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package codegen

import "path"

// GroupOptions contains options for generating a version client
type GroupOptions struct {
	Location  Location
	Package   Package
	Group     string
	Version   string
	Types     GroupTypes
	Names     GroupNames
	Resources map[string]*ResourceOptions
}

// GroupTypes contains types for generating a version client
type GroupTypes struct {
	Interface string
	Struct    string
}

// GroupNames contains names for generating a version client
type GroupNames struct {
	Natural string
	Proper  string
}

func generateVersionClient(options GroupOptions) error {
	if err := generateTemplate(getTemplate("groupclient.tpl"), path.Join(options.Location.Path, options.Location.File), options); err != nil {
		return err
	}

	for _, resource := range options.Resources {
		if err := generateResourceReader(*resource); err != nil {
			return err
		}
		if err := generateResourceClient(*resource); err != nil {
			return err
		}
		if err := generateResourceReference(*resource); err != nil {
			return err
		}
		if err := generateResource(*resource); err != nil {
			return err
		}
	}
	return nil
}
