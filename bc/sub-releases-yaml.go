//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

package bc

import (
	"fmt"

	"github.com/mrmxf/util/kfg"
	"github.com/spf13/cobra"
)

// releaseYamlCmd prints the path to the releases YAML file
var releaseYamlCmd = &cobra.Command{
	Use:           "yaml",
	SilenceErrors: true,
	SilenceUsage:  true,
	Short:         "Print the releases YAML file path",
	Long:          `Print the path to the releases YAML file used for version tracking.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(kfg.ReleasesPath())
	},
}

func init() {
	// Add yaml subcommand to the releases command
	releasesCmd.AddCommand(releaseYamlCmd)
}
