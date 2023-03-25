// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"github.com/onosproject/helmit/internal/logging"
	"math/rand"
	"time"

	"github.com/spf13/cobra"
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
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			verbose, _ := cmd.Flags().GetBool("verbose")
			logging.SetVerbose(verbose)
			return nil
		},
	}
	cmd.AddCommand(getTestCommand())
	cmd.AddCommand(getBenchCommand())
	cmd.PersistentFlags().BoolP("verbose", "v", false, "enable verbose output")
	return cmd
}
