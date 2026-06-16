//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

package buildinfo

import (
	"fmt"
	"log/slog"
	"runtime"

	"github.com/spf13/cobra"
)

var flagShort bool
var flagVersion bool

// Command prints build and version information.
// Use "version" as an alias: `clog version` == `clog buildinfo -v`
var Command = &cobra.Command{
	Use:     "buildinfo",
	Aliases: []string{"version"},
	Short:   "Show build and version information",
	Long: `buildinfo displays version information in various formats.

By default it prints all build metadata. Use -v for just the semantic version
or --version for the long human-readable version string.

Examples:
  clog buildinfo             # all build metadata
  clog buildinfo -v          # v1.2.3 only
  clog buildinfo --version   # long human-readable string
  clog version               # alias for buildinfo -v`,

	Run: func(cmd *cobra.Command, args []string) {
		info := Info()
		switch {
		case flagShort:
			fmt.Println(info.Short)
		case flagVersion:
			fmt.Println(info.Long)
		default:
			fmt.Printf("App:     %s (%s)\n", info.AppTitle, info.AppName)
			fmt.Printf("Version: %s\n", info.Short)
			fmt.Printf("Long:    %s\n", info.Long)
			if info.Tag != "" {
				fmt.Printf("Tag:     %s\n", info.Tag)
			}
			if info.CommitId != "" {
				fmt.Printf("Commit:  %s\n", info.CommitId)
			}
			fmt.Printf("Date:    %s\n", info.Date)
			fmt.Printf("OS:      %s/%s\n", info.OS, info.ARCH)
			if info.Note != "" {
				fmt.Printf("Note:    %s\n", info.Note)
			}
		}
	},
}

func init() {
	_, file, _, _ := runtime.Caller(0)
	slog.Debug("init " + file)
	Command.Flags().BoolVarP(&flagShort, "v", "v", false, "short semantic version only (e.g. v1.2.3)")
	Command.Flags().BoolVar(&flagVersion, "version", false, "long human-readable version string")
}
