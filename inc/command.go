//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

package inc

import (
	"embed"
	"fmt"
	"io"
	"log/slog"
	"os"
	"runtime"

	"github.com/mrmxf/util/crayon"
	"github.com/spf13/cobra"
)

var CommandString = "Inc"

// IncFs is the file system for the embedded shell file - override at runtime to use your own
//
//go:embed inc.sh
var IncFs embed.FS

var DarkMode bool = false
var JustCrayon bool = false

// Command outputs embedded helper script to stdout.
// This command sends embedded shell scripts and color helpers to stdout for sourcing in scripts.
var Command = &cobra.Command{
	Use:   CommandString,
	Short: "Send embedded helper script to stdout",
	Long: `Inc outputs embedded helper scripts and color definitions to stdout.

This command is designed to be sourced in shell scripts to provide common
utilities and color variables. It automatically includes color definitions
followed by the embedded helper script.

The output can be sourced directly in bash/zsh scripts:
  eval "$(clog Inc)"           # Source helpers and colors
  source <(clog Inc)           # Alternative sourcing method

The command always outputs color definitions first, followed by the
embedded shell helper functions.

Examples:
  clog Inc                     # Output helpers with normal colors
  clog Inc -D                  # Output helpers with dark mode colors
  eval "$(clog Inc)" && myFunc # Source and use helper functions

Returns exit status 126 if embedded file not found.`,
	Run: func(cmd *cobra.Command, args []string) {

		src, err := IncFs.Open("inc.sh")
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
			os.Exit(126)
		}
		defer src.Close()

		// start with the color helpers
		fmt.Println(crayon.GetBashString(DarkMode))

		dst := os.Stdout
		nBytes, err := io.Copy(dst, src)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s (%d bytes copied)\n", err, nBytes)
			os.Exit(126)
		}
	},
}

func init() {
	_, file, _, _ := runtime.Caller(0)
	slog.Debug("init " + file)
	Command.PersistentFlags().BoolVarP(&DarkMode, "darkmode", "D", false, "all colors for darkmode")
}
