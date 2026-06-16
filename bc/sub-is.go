//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

package bc

import (
	"log/slog"
	"os"

	"github.com/spf13/cobra"
)

// isCmd compares a field from the first release with a provided string value
var isCmd = &cobra.Command{
	Use:   "is",
	Short: "Compare a release field with a string value",
	Long: `Compare a field from the first release with a provided string value.
Usage: clog bc is <field> <value>

Fields: version, date, flow, build, note

Exit codes:
- 0: values match
- 1: values don't match or error occurred`,
	Args:          cobra.ExactArgs(2),
	SilenceErrors: true,
	SilenceUsage:  true,
	Run: func(cmd *cobra.Command, args []string) {
		if len(Releases()) == 0 {
			slog.Debug("is command: no release data available")
			slog.Error("no release data available")
			os.Exit(1)
		}

		field := args[0]
		expected := args[1]
		release := Releases()[0]

		var actual string
		switch field {
		case "version":
			actual = release.Version
		case "date":
			actual = release.Date
		case "flow":
			actual = release.Flow
		case "build":
			actual = release.Build
		case "note":
			actual = release.Note
		default:
			slog.Debug("is command: invalid field", "field", field, "valid_fields", "version,date,flow,build,note")
			slog.Error("invalid field", "field", field, "valid_fields", "version, date, flow, build, note")
			os.Exit(1)
		}

		slog.Debug("is command comparison", "field", field, "actual", actual, "expected", expected, "match", actual == expected)

		if actual == expected {
			os.Exit(0)
		} else {
			os.Exit(1)
		}
	},
}

func init() {
	// Add is subcommand to the main BC command
	Command.AddCommand(isCmd)
}
