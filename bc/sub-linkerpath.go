//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

package bc

import (
	"fmt"

	semver "github.com/mrmxf/util/buildinfo"
	"github.com/spf13/cobra"
)

// linkerPathCmd prints the dynamic linker path for SemVerJSON
var linkerPathCmd = &cobra.Command{
	Use:           "linkerpath",
	SilenceErrors: true,
	SilenceUsage:  true,
	Short:         "Print the dynamic linker path for SemVerJSON",
	Long: `Print the linker path for the SemVerJSON variable that would be used with go build -ldflags.
This path is dynamically discovered at runtime, equivalent to:
go tool objdump -S <binary> | grep 'semver.SemVerJSON'`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(semver.LinkerPath())
	},
}

func init() {
	// Add linkerpath subcommand to the main BC command
	Command.AddCommand(linkerPathCmd)
}
