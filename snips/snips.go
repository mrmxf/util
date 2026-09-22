//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

//
// package snips provide handling functions to enable snippets

package snips

import (
	"fmt"
	"log/slog"
	"os"
	"runtime"

	"github.com/mrmxf/util/scripts"
	"github.com/spf13/cobra"
)

// create helper types
type Snippet string
type SnippetGroup map[Snippet]any
type RawSnippets map[string]any

// a parsed snippets struct
type ParsedSnippets struct {
	Snippets  SnippetGroup
	ParentCmd *cobra.Command // the parent command of the snippet group
}

type ListSnippetsData struct {
	Title   string
	Key     string
	Parsed  *ParsedSnippets
	Verbose bool
	Plain   bool
}

// add snippets to the main list of root commands, found with given key
func ParseSnippets(rootCmd *cobra.Command, raw RawSnippets) (ParsedSnippets, error) {
	pSnips := ParsedSnippets{
		Snippets:  SnippetGroup{},
		ParentCmd: rootCmd,
	}

	recurseRawMap(pSnips.ParentCmd, pSnips.Snippets, 0, raw)

	return pSnips, nil
}

func kmdPath(c *cobra.Command, kmd string) string {
	return fmt.Sprintf("%s %s", c.CommandPath(), kmd)
}

// overridesKey is the annotation holding the path of the built-in command that
// a snippet replaced.
const overridesKey = "overrides"

// addSnippetCmd adds a snippet command to parentCmd. If parentCmd already has a
// built-in (non-snippet) command of the same name, the built-in is removed so
// the local snippet deterministically wins, and the replaced command's path is
// recorded so warnIfOverride can report it when the snippet runs.
func addSnippetCmd(parentCmd, cmd *cobra.Command) {
	for _, existing := range parentCmd.Commands() {
		if existing.Name() == cmd.Name() && existing.Annotations["is-a"] != "snippet" {
			cmd.Annotations[overridesKey] = existing.CommandPath()
			slog.Debug("snippet overrides built-in command", "command", existing.CommandPath())
			parentCmd.RemoveCommand(existing)
			break
		}
	}
	parentCmd.AddCommand(cmd)
}

// wantsHelp reports that the caller asked for the snippet's help rather than
// for it to run. Snippets parse their own flags, so this is the one cobra flag
// still honoured, and only in first position - `clog build --help` explains the
// command, while `clog build dev --help` is the script's business.
func wantsHelp(args []string) bool {
	return len(args) > 0 && (args[0] == "--help" || args[0] == "-h")
}

// warnIfOverride warns when the snippet being run (or a snippet group it lives
// in) replaced a built-in command.
func warnIfOverride(cmd *cobra.Command) {
	for c := cmd; c != nil; c = c.Parent() {
		if builtin := c.Annotations[overridesKey]; builtin != "" {
			slog.Warn("overriding built-in command", "snippet", cmd.CommandPath(), "built-in", builtin)
			return
		}
	}
}

func recurseRawMap(parentCmd *cobra.Command, group SnippetGroup, depth int, raw RawSnippets) {
	for kmd, snip := range raw {
		switch skript := snip.(type) {

		case int:
			// an int is a valid ZSH / BASH command !!
			slog.Debug(fmt.Sprintf("%d.leaf - %s %T", depth, kmd, skript))
			cmd := &cobra.Command{
				Use:   kmd,
				Short: kmdPath(parentCmd, kmd),
				// A snippet is a shell script: its flags are its own, and it
				// needs the raw argv. Whitelisting unknown flags stops the
				// error but cobra still consumes them, so `clog build dev
				// --fast` would reach the script as `dev`. help is intercepted
				// in Run so `clog <snippet> --help` still works.
				DisableFlagParsing: true,
				Annotations: map[string]string{
					"command": kmdPath(parentCmd, kmd),
					"depth":   fmt.Sprintf("%d", depth),
					"is-a":    "snippet",
					"script":  fmt.Sprintf("%d", skript),
					"type":    fmt.Sprintf("%T", skript),
				},
				Run: func(cmd *cobra.Command, args []string) {
					if wantsHelp(args) {
						cmd.Help()
						return
					}
					warnIfOverride(cmd)
					ident := fmt.Sprintf("snippet: %s", cmd.CommandPath())
					strInt := fmt.Sprintf("%d", skript)
					slog.Debug(fmt.Sprintf("snippet: %s\n$ %s\n", kmd, strInt))
					exitStatus, err := scripts.AwaitShellSnippet(strInt, nil, args)
					if err != nil {
						slog.Error("failed to stream snippet "+ident, "error", err)
					}
					os.Exit(exitStatus)
				},
			}
			addSnippetCmd(parentCmd, cmd)
			group[Snippet(kmd)] = skript

		case string:
			slog.Debug(fmt.Sprintf("%d.leaf - %s %T", depth, kmd, skript))
			cmd := &cobra.Command{
				Use:   kmd,
				Short: "snippet " + kmdPath(parentCmd, kmd),
				// A snippet is a shell script: its flags are its own, and it
				// needs the raw argv. Whitelisting unknown flags stops the
				// error but cobra still consumes them, so `clog build dev
				// --fast` would reach the script as `dev`. help is intercepted
				// in Run so `clog <snippet> --help` still works.
				DisableFlagParsing: true,
				Annotations: map[string]string{
					"command": kmdPath(parentCmd, kmd),
					"depth":   fmt.Sprintf("%d", depth),
					"is-a":    "snippet",
					"script":  skript,
					"type":    fmt.Sprintf("%T", skript),
				},
				Run: func(cmd *cobra.Command, args []string) {
					if wantsHelp(args) {
						cmd.Help()
						return
					}
					warnIfOverride(cmd)
					ident := fmt.Sprintf("snippet: %s", cmd.CommandPath())
					slog.Debug(fmt.Sprintf("snippet: %s\n$ %s\n", ident, skript))
					exitStatus, err := scripts.AwaitShellSnippet(skript, nil, args)
					if err != nil {
						slog.Error("failed to stream snippet "+ident, "error", err)
					}
					os.Exit(exitStatus)
				},
			}
			addSnippetCmd(parentCmd, cmd)
			group[Snippet(kmd)] = skript

		case map[string]interface{}:
			slog.Debug(fmt.Sprintf("%d.node - %s %T", depth, kmd, snip))
			cmd := &cobra.Command{
				Use:   kmd,
				Short: kmdPath(parentCmd, kmd),
				Annotations: map[string]string{
					"command": kmdPath(parentCmd, kmd),
					"depth":   fmt.Sprintf("%d", depth),
					"is-a":    "snippet",
					"script":  kmdPath(parentCmd, kmd) + " --help",
					"type":    "node",
				},
				Run: func(cmd *cobra.Command, args []string) {
					warnIfOverride(cmd)
					cmd.Help()
					os.Exit(1)
				},
			}
			// add this command stub to the tree and descend
			addSnippetCmd(parentCmd, cmd)
			newGroup := SnippetGroup{}
			group[Snippet(kmd)] = &newGroup
			recurseRawMap(cmd, newGroup, depth+1, snip.(map[string]interface{}))
		default:
			slog.Debug(fmt.Sprintf("%d.WARNING ignoring unexpected snippet (%s) of type %s", depth, kmd, snip))
		}
	}
}

func init() {
	// log the order of the init files in case there are problems
	_, file, _, _ := runtime.Caller(0)
	slog.Debug("init " + file)
}
