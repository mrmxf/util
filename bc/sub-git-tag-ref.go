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

// GitTagRef returns the current version reference from releases
// Note that it is the responsibility of the manager of releases.yaml to add or remove a 'v' prefix
func GitTagRef() string {
	if len(Releases()) == 0 {
		return ""
	}

	return Releases()[0].Version
}

// refCmd prints the current version number in releases.yaml
var refCmd = &cobra.Command{
	Use:           "ref",
	SilenceErrors: true,
	SilenceUsage:  true,
	Short:         "Print the current version reference",
	Long:          `Print the current version reference from releases, with 'v' prefix if go.mod exists.`,
	Run: func(cmd *cobra.Command, args []string) {
		version := GitTagRef()
		if version == "" {
			slog.Error("no release data available")
			os.Exit(1)
		}

		fmt.Println(version)
	},
}

func init() {
	// Add ref subcommand to the tag command
	tagCmd.AddCommand(refCmd)
}
