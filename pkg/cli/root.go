// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/onosproject/helmit/pkg/util/logging"
	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
}

// GetRootCommand returns the root helmit command
func GetRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "helmit <command> [args]",
		Short:        "Setup test clusters and run integration tests on Kubernetes",
		SilenceUsage: true,
	}
	cmd.AddCommand(getTestCommand())
	cmd.AddCommand(getBenchCommand())
	cmd.AddCommand(getSimulateCommand())
	cmd.PersistentFlags().BoolP("verbose", "v", false, "Enable verbose output")
	return cmd
}

// GenerateCliDocs generate markdown files for helmit commands
func GenerateCliDocs() {
	cmd := GetRootCommand()
	err := doc.GenMarkdownTree(cmd, "docs/cli")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

}

func setupCommand(cmd *cobra.Command) {
	verbose, _ := cmd.Flags().GetBool("verbose")
	logging.SetVerbose(verbose)
}
