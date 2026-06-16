//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

package bc

import (
	"github.com/spf13/cobra"
)

// releasesCmd provides BC (build-control) releases operations and management.
// It prints help when no subcommands are given.
var releasesCmd = &cobra.Command{
	Use:           "releases",
	SilenceErrors: true,
	SilenceUsage:  true,
	Short:         "BC (build-control) releases operations",
	Long: `BC (build-control) releases operations provide comprehensive release management:
- View release configuration paths
- Access release information
- Manage release metadata

Use 'clog bc releases <command> --help' for more information about specific release commands.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Print help when no subcommands are provided
		cmd.Help()
	},
}

func init() {
	// Add releases subcommand to the main BC command
	Command.AddCommand(releasesCmd)
}
