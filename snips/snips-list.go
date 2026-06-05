//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

// package snips provide handling functions to enable snippets

package snips

import (
	"fmt"
	"log/slog"
	"sort"
	"strings"
)

func ListSnippets(d *ListSnippetsData) {
	fmt.Println(">>>" + d.Title + " in config key `" + d.Key + "`")

	if d.Parsed == nil || len(d.Parsed.Snippets) == 0 {
		slog.Warn("No " + d.Title + " found with  config key `" + d.Key + "`")
		return
	}

	recurseListMap(d.Parsed.ParentCmd.CommandPath(), d.Parsed.Snippets, d, 0)
}

// make a recursive function to sort & list all the snippets
func recurseListMap(cmdPath string, sMap SnippetGroup, d *ListSnippetsData, depth int) {
	keys := make([]string, 0, len(sMap))
	for k := range sMap {
		keys = append(keys, string(k))
	}
	sort.Strings(keys)
	pad := strings.Repeat(" ", depth*2)

	for _, k := range keys {
		// snip is either the command string or a subcommand
		snip := sMap[Snippet(k)]

		// we always print the name of the command (or sub command)
		plainKmd := fmt.Sprintf("%s  %s %s", pad, cmdPath, k)
		switch snipType := snip.(type) {
		case int:
			if d.Verbose {
				plainKmd = fmt.Sprintf("%s\n%s   %d", plainKmd, pad, snip)
			}
			fmt.Println(plainKmd)
		case *SnippetGroup:
			fmt.Println("+", plainKmd)
			recurseListMap(cmdPath+" "+k, *snip.(*SnippetGroup), d, depth+1)
		case string:
			if d.Verbose {
				plainKmd = fmt.Sprintf("%s\n%s   %s", plainKmd, pad, snip)
			}
			fmt.Println(plainKmd)
		default:
			slog.Error("WTF - (%T) %s %s", snipType, cmdPath, k)
		}
	}
	if depth > 0 {
		fmt.Println(strings.Repeat("-", 80))
	}
}
