// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package codegen

import "path"

// ClientOptions contains options for generating a client
type ClientOptions struct {
	Location Location
	Package  Package
	Types    ClientTypes
	Groups   map[string]*GroupOptions
}

// ClientTypes contains types for generating a client
type ClientTypes struct {
	Interface string
	Struct    string
}

func generateClient(options ClientOptions) error {
	if err := generateTemplate(getTemplate("client.tpl"), path.Join(options.Location.Path, options.Location.File), options); err != nil {
		return err
	}

	for _, group := range options.Groups {
		if err := generateVersionClient(*group); err != nil {
			return err
		}
	}
	return nil
}
