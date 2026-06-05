//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

// package cmd.snippets lists the available snippets commands
package snippets

import (
	"fmt"
	"log/slog"
	"runtime"

	"github.com/mrmxf/util/snips"
	"github.com/spf13/cobra"
)

type Command = struct {
	Use     string            // the actual command
	Title   string            //display title for listing the snippets
	Key     string            // the default key used to find snippets
	Verbose bool              // list verbose or short
	Plain   bool              // list as plain or pretty colors
	Raw     snips.RawSnippets // the raw (parsed yaml) snippets
	Cmd     *cobra.Command
}

// Bootstrap returns a pointer to a new cobra command for the CLI
// You can have multiple snippets branches in multiple files and put them
// in the CLI hierarchy any way you like.
func Bootstrap(parentCmd *cobra.Command, opts Command) *cobra.Command {
	Snippets, err := snips.ParseSnippets(parentCmd, opts.Raw)
	if err != nil {
		slog.Error("error parsing snippets", "error", err)
	}

	var Command = &cobra.Command{
		Use:   opts.Use,
		Short: "list snippets found in the config key " + opts.Key,
		Long:  `local config adds & overwrites the core snippets`,

		Run: func(cmd *cobra.Command, args []string) {
			snips.ListSnippets(&snips.ListSnippetsData{
				Title:   opts.Title,
				Key:     opts.Key,
				Parsed:  &Snippets,
				Verbose: opts.Verbose,
				Plain:   opts.Plain,
			})

			if !opts.Verbose {
				fmt.Println("\nclog Snippets --show   # show full shell snippet strings")
			}
		},
	}
	Command.PersistentFlags().BoolVarP(&opts.Verbose, "verbose", "V", false, "clog Snippets -v   # verbose scripts")
	Command.PersistentFlags().BoolVarP(&opts.Plain, "plain", "P", false, "clog Snippets -p   # remove pretty colors")
	return Command
}

func init() {
	// log the order of the init files in case there are problems
	_, file, _, _ := runtime.Caller(0)
	slog.Debug("init " + file)

}
