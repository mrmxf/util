//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

package should

import (
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"strings"

	"github.com/mrmxf/util/slogger"
	"github.com/spf13/cobra"
)

var debug bool

// Command checks if an environment variable contains a space-delimited word.
// This command provides logic helper functionality for bash scripts.
var Command = &cobra.Command{
	SilenceErrors: true,
	SilenceUsage:  true,
	Use:           "Should <env_var> <word>",
	Short:         "Check if environment variable contains a space-delimited word",
	Long: `Should checks if an environment variable contains a specific word in its space-delimited value.

This command is designed as a logic helper for bash scripts, providing exit codes
that can be used in conditional statements:
  - Exit code 0: Word found in environment variable
  - Exit code 1: Word not found in environment variable
  - Exit code 2: Environment variable does not exist

This is particularly useful for checking build targets, configuration flags,
or any space-separated lists stored in environment variables.

Examples:
  export MAKE="data hugo exe ko"
  clog Should MAKE "hugo"         # Returns 0 (success) - hugo found
  clog Should MAKE "docker"       # Returns 1 (not found) - docker missing
  clog Should MISSING "anything"  # Returns 2 (no env var)

  # In shell scripts:
  if clog Should TARGETS "deploy"; then
    echo "Deploy target is enabled"
  fi`,
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		if debug {
			slogger.UsePrettyLogger(slog.LevelDebug)
		}
		needle := args[1]
		haystackEnv := args[0]
		haystack, envExists := os.LookupEnv(haystackEnv)
		if envExists {
			words := strings.Split(haystack, " ")
			found := false
			for i := range words {
				if words[i] == needle {
					found = true
				}
			}
			if found {
				dbg := fmt.Sprintf("clog Should $%s \"%s\" - ✅ found in(%s)", haystackEnv, needle, haystack)
				slog.Debug(dbg)
				os.Exit(0)
			} else {
				dbg := fmt.Sprintf("clog Should $%s \"%s\" ❌ missing from(%s)", haystackEnv, needle, haystack)
				slog.Debug(dbg)
				os.Exit(1)
			}
		} else {
			dbg := fmt.Sprintf("clog Should $%s \"%s\" - ❌ missing env %s", haystackEnv, needle, haystackEnv)
			slog.Debug(dbg)
			os.Exit(2)
		}
	},
}

func init() {
	_, file, _, _ := runtime.Caller(0)
	slog.Debug("init " + file)

	Command.PersistentFlags().BoolVarP(&debug, "debug", "D", false, "clog Should -D $MAKE \"thingy\"")
}
