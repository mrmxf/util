//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

package bc

import (
	"github.com/spf13/cobra"
)

// checkoutCmd provides BC (build-control) Git checkout operations and management.
// It prints help when no subcommands are given.
var checkoutCmd = &cobra.Command{
	Use:           "checkout",
	SilenceErrors: true,
	SilenceUsage:  true,
	Short:         "BC (build-control) Git checkout operations",
	Long: `BC (build-control) Git checkout operations provide comprehensive checkout management:
- Checkout branches, tags, or commits
- List checkout-related information
- Validation and verification

Use 'clog bc git checkout <command> --help' for more information about specific checkout commands.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Print help when no subcommands are provided
		cmd.Help()
	},
}

func init() {
	// Add checkout subcommand to the git command
	gitCmd.AddCommand(checkoutCmd)
}
