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

// releaseVersionCmd prints the version from the first release
var releaseVersionCmd = &cobra.Command{
	Use:           "version",
	SilenceErrors: true,
	SilenceUsage:  true,
	Short:         "Print the release version",
	Long:          `Print the version from the first release in the releases configuration.`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(Releases()) == 0 {
			slog.Error("no release data available")
			os.Exit(1)
		}

		fmt.Println(Releases()[0].Version)
	},
}

func init() {
	// Add version subcommand to the releases command
	releasesCmd.AddCommand(releaseVersionCmd)
}
