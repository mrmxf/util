//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

//
// package source copies a snippet or script file to stdout
//
// for example, a snippet in `clog.yaml` or a script file like
//  `hello.sh` can be run in a new shell `clog hello`. To source it  in the
//  current shell ENV values in the current context) then use `eval
// "$(clog Source hello)"` which is more portable than `source <(clog Source
// hello)`.

package source

import (
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

var CommandString = "Source"

var SnippetsTree bool = false
var JustCrayon bool = false

// Command define the cobra settings for this command
var Command = &cobra.Command{
	Use:   CommandString,
	Short: "source a script or snippet",
	Long:  `if it runs with clog cmd then it will source with eval "$(clog Source cmd)"`,
	Run: func(cmd *cobra.Command, args []string) {
		cmdString := strings.Join(args, " ")

		slog.Debug("searching snippets:  " + cmdString)
		srcCmd, err := FindSnippet(cmd.Root(), args)
		if err == nil {
			// source the snippet or script to stdout
			switch srcCmd.Annotations["is-a"] {
			case "snippet":
				fmt.Println(srcCmd.Annotations["script"])
				os.Exit(0)
			case "script":
				fmt.Println(srcCmd.Annotations["file-path"])
				os.Exit(0)
			default:
				slog.Error(fmt.Sprintf("clog Source (%s) neither snippet nor script", cmdString))
				os.Exit(1)
			}
		}
		slog.Error(fmt.Sprintf("clog Source (%s) %s", cmdString, err.Error()))
		os.Exit(1)
	},
}

func init() {
	_, file, _, _ := runtime.Caller(0)
	slog.Debug("init " + file)
}
