//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/

package snips

import (
	"strings"

	"github.com/spf13/cobra"
)

// ResolveCase rewrites argv so a command typed in the wrong case still runs.
//
// The case of a command name carries meaning in clog: a Capitalised name is
// shipped (compiled in, or from util's embedded konfig) and a lowercase name
// is yours, from a .clog.yaml. Where both exist they are two different
// commands on purpose - `clog build` runs your override and `clog Build` runs
// the shipped one it overrides, so an override can be A/B'd against its
// default instead of replacing it blind. ParseSnippets already keeps that pair
// distinct; this function is the other half of the bargain, which is that
// nobody should have to hold shift when there is no ambiguity to resolve.
//
// Resolution, at each level of the command tree:
//
//  1. exact match wins - this is what preserves the A/B pair
//  2. otherwise a unique case-insensitive match is rewritten to its real name
//  3. otherwise the argument is left alone for cobra to reject
//
// Cobra's own EnableCaseInsensitive is deliberately not used: it is global and
// matches case-insensitively at step 1, so `build` and `Build` would collide
// and resolve by registration order. That would quietly destroy the pair the
// whole convention exists to provide.
//
// When a fold is ambiguous - `BUILD` where both `Build` and `build` exist -
// the shipped command wins, because the promise attached to the Capitalised
// name is that it is always reachable.
func ResolveCase(root *cobra.Command, args []string) []string {
	out := make([]string, len(args))
	copy(out, args)

	cmd := root
	for i, arg := range out {
		// Flags and their values end the command path; everything after the
		// first one belongs to the command, not to the tree.
		if strings.HasPrefix(arg, "-") {
			break
		}
		match := findChild(cmd, arg)
		if match == nil {
			break
		}
		out[i] = match.Name()
		cmd = match
	}
	return out
}

// findChild returns the child of cmd that arg names, exact match first.
func findChild(cmd *cobra.Command, arg string) *cobra.Command {
	// 1. exact - on the name or any alias
	for _, c := range cmd.Commands() {
		if c.Name() == arg {
			return c
		}
		for _, a := range c.Aliases {
			if a == arg {
				return c
			}
		}
	}
	// 2. unique case-insensitive fold, preferring a shipped (Capitalised) name
	var folded []*cobra.Command
	for _, c := range cmd.Commands() {
		if strings.EqualFold(c.Name(), arg) {
			folded = append(folded, c)
			continue
		}
		for _, a := range c.Aliases {
			if strings.EqualFold(a, arg) {
				folded = append(folded, c)
				break
			}
		}
	}
	switch len(folded) {
	case 0:
		return nil
	case 1:
		return folded[0]
	default:
		for _, c := range folded {
			if isShipped(c) {
				return c
			}
		}
		return folded[0]
	}
}

// isShipped reports whether a command name is the Capitalised half of a pair.
// The test is the first rune rather than an annotation so it holds for
// compiled-in commands and konfig snippets alike.
func isShipped(c *cobra.Command) bool {
	n := c.Name()
	if n == "" {
		return false
	}
	r := []rune(n)[0]
	return r >= 'A' && r <= 'Z'
}
