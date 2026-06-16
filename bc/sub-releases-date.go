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

// releaseDateCmd prints the date from the first release
var releaseDateCmd = &cobra.Command{
	Use:           "date",
	SilenceErrors: true,
	SilenceUsage:  true,
	Short:         "Print the release date",
	Long:          `Print the date from the first release in the releases configuration.`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(Releases()) == 0 {
			slog.Error("no release data available")
			os.Exit(1)
		}

		fmt.Println(Releases()[0].Date)
	},
}

func init() {
	// Add date subcommand to the releases command
	releasesCmd.AddCommand(releaseDateCmd)
}
