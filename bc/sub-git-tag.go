//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

package bc

import (
	"github.com/spf13/cobra"
)

// tagCmd provides BC (build-control) Git tag operations and management.
// It prints help when no subcommands are given.
var tagCmd = &cobra.Command{
	Use:           "tag",
	SilenceErrors: true,
	SilenceUsage:  true,
	Short:         "BC (build-control) Git tag operations",
	Long: `BC (build-control) Git tag operations provide comprehensive tag management:
- View current HEAD tag
- Create new tags
- List existing tags
- Tag validation and verification

Use 'clog bc git tag <command> --help' for more information about specific tag commands.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Print help when no subcommands are provided
		cmd.Help()
	},
}

func init() {
	// Add tag subcommand to the git command
	gitCmd.AddCommand(tagCmd)
}
