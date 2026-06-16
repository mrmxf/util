//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

package bc

import (
	"fmt"
	"os"
	"runtime"

	slog "github.com/mrmxf/util/slogger"

	"github.com/spf13/cobra"
)

// stashPathCmd prints the stash file path
var stashPathCmd = &cobra.Command{
	Use:           "path",
	SilenceErrors: true,
	SilenceUsage:  true,
	Short:         "Print the stash file path",
	Long:          "Prints the absolute path to the stash file used for BC flow error tracking",
	Run:           stashPathRun,
}

func init() {
	_, file, _, _ := runtime.Caller(0)
	slog.Debug("init " + file)

	// Add path subcommand to the stash get command
	stashGetCmd.AddCommand(stashPathCmd)
}

// stashPathRun prints the stash file path or logs an error if not configured
func stashPathRun(cmd *cobra.Command, args []string) {
	stashPath := getStashPath()

	if len(stashPath) > 0 {
		fmt.Println(stashPath)
	} else {
		slog.Error("clog.stash-path is not defined in konfig.yaml")
		os.Exit(1)
	}
}
