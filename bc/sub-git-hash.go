//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

package bc

import (
	"github.com/spf13/cobra"
)

// hashCmd provides BC (build-control) Git hash operations and management.
// It prints help when no subcommands are given.
var hashCmd = &cobra.Command{
	Use:           "hash",
	SilenceErrors: true,
	SilenceUsage:  true,
	Short:         "BC (build-control) Git hash operations",
	Long: `BC (build-control) Git hash operations provide comprehensive hash management:
- View current HEAD hash
- Get commit hashes
- Hash validation and verification
- Short hash operations

Use 'clog bc git hash <command> --help' for more information about specific hash commands.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Print help when no subcommands are provided
		cmd.Help()
	},
}

func init() {
	// Add hash subcommand to the git command
	gitCmd.AddCommand(hashCmd)
}
