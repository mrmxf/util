//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

package bc

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"
)

// releaseBuildCmd prints the build from the first release
var releaseBuildCmd = &cobra.Command{
	Use:           "build",
	SilenceErrors: true,
	SilenceUsage:  true,
	Short:         "Print the release build",
	Long:          `Print the build from the first release in the releases configuration.`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(Releases()) == 0 {
			slog.Error("no release data available")
			os.Exit(1)
		}

		fmt.Println(Releases()[0].Build)
	},
}

func init() {
	// Add build subcommand to the releases command
	releasesCmd.AddCommand(releaseBuildCmd)
}
