//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

package bc

import (
	"github.com/spf13/cobra"
)

// BCCmd is the main BC (build-control) command that provides build control functionality.
// It prints help when no subcommands are given.
var Command = &cobra.Command{
	Use:   "BC",
	Short: "BC <sub-command> - clog's Build Control and version management",
	Long:  longHelp,
	Run: func(cmd *cobra.Command, args []string) {
		// Print help when no subcommands are provided
		cmd.Help()
	},
}
