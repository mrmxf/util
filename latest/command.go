//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

package latest

import (
	"github.com/spf13/cobra"
)

// Command is the main Latest command that provides latest version information for packages.
// It prints help when no subcommands are given.
var Command = &cobra.Command{
	Use:   "Latest",
	Short: "Latest <package> - get the latest version information for packages",
	Long: `Latest provides latest version information for various packages including:
- TinyGo compiler releases
- Package version strings formatted for deployment
- Platform-specific package information
- GitHub releases API integration

Use 'clog Latest <package> --help' for more information about specific packages.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Print help when no subcommands are provided
		cmd.Help()
	},
}
