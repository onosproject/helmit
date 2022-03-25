// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"os"

	"github.com/onosproject/helmit/pkg/cli"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

func main() {
	cmd := cli.GetRootCommand()
	if err := cmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
