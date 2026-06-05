//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

package crayon

import (
	"fmt"
	"log/slog"
	"runtime"

	"github.com/spf13/cobra"
)

var hideComment bool
var hideShellScript bool
var showExample bool
var DarkMode bool

// Command provides bash color string for scripts.
// Allows scripts to access the same CLI colors as clog.
//
// The default case prints a string for bash & zsh & sh to have colors:
//
//	source <(clog Crayon)
//	printf "pretty $cE red error $cI yellow info $cW magenta warning $cX\n"
var Command = &cobra.Command{
	Use:   "Crayon",
	Short: "Get bash string for default colors",
	Long: `Crayon provides bash-compatible color environment variables for scripts.

This command outputs shell variable definitions that can be sourced to enable
colored output in bash, zsh, and sh scripts. The color scheme matches the
colors used throughout clog.

Usage patterns:
  source <(clog Crayon)           # Source colors into current shell
  eval "$(clog Crayon)"           # Alternative sourcing method
  clog Crayon -S > colors.sh      # Save to file for later use

Available color variables after sourcing:
  $cE - Error (red)      $cI - Info (yellow)     $cF - File (cyan)
  $cH - Header (bold)    $cS - Success (green)   $cT - Text (normal)
  $cU - URL (underline)  $cW - Warning (magenta) $cX - Reset colors

Examples:
  clog Crayon                     # Show comment and script
  clog Crayon -C                  # Comment only
  clog Crayon -S                  # Script only
  clog Crayon -E                  # Show color examples`,
	Run: func(cmd *cobra.Command, args []string) {
		comment := "# Command  | Error     | Info      | File      | Header      | Success   | Text      | Url       | Warning   | eXit     | AMD64         | Arm64       | Linux       | Mac         | Windows\n"
		bashstr := GetBashString(false)

		if !hideComment {
			fmt.Print(comment)
		}
		if !hideShellScript {
			fmt.Print(bashstr + "\n")
		}
		if showExample {
			fmt.Println(SampleColors())
		}
	},
}

func init() {
	Command.PersistentFlags().BoolVarP(&hideShellScript, "comment", "C", false, "print the comment, no Shell script")
	Command.PersistentFlags().BoolVarP(&hideComment, "script", "S", false, "print the Shell script, no comment")
	Command.PersistentFlags().BoolVarP(&showExample, "example", "E", false, "show an example of colors")
	Command.PersistentFlags().BoolVarP(&DarkMode, "darkmode", "D", false, "all colors for darkmode")
	_, file, _, _ := runtime.Caller(0)
	slog.Debug("init " + file)
}
