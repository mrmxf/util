//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

package bc

import (
	"github.com/spf13/cobra"
)

// gitCmd provides BC (build-control) Git operations and repository management.
// It prints help when no subcommands are given.
var gitCmd = &cobra.Command{
	Use:           "git",
	SilenceErrors: true,
	SilenceUsage:  true,
	Short:         "BC (build-control) Git operations",
	Long: `BC (build-control) Git operations provide comprehensive Git repository management:
- Tag creation and management
- Repository status and information
- Branch operations
- Commit and history management

Use 'clog bc git <command> --help' for more information about specific Git commands.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Print help when no subcommands are provided
		cmd.Help()
	},
}

func init() {
	// Add git subcommand to the main BC command
	Command.AddCommand(gitCmd)
}
