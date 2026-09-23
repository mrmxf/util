//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/

// Package retire keeps a renamed command reachable as an error.
//
// clog v1.0.0 settled a grammar: the verb says what you get back, so `get`
// returns one value, `list` returns many, `show` returns a document and
// `is`/`has`/`should`/`require` return only an exit code. Commands that broke
// that rule were renamed rather than aliased, because an alias leaves the old
// spelling working and the inconsistency then lives forever in scripts and in
// shell history.
//
// A retired command is registered anyway, and always fails. It never does the
// old thing - a command that silently still works is how a rename half-lands.
// The error names the replacement, so the fix is the next thing you type:
//
//	$ clog CI policy
//	Error: `clog CI policy` was retired in v1.0.0
//	  why: a bare noun says nothing about whether it prints a value, a list or a document
//	  use: clog CI show policy
//
// These are permanent. They cost one cobra command each and they are the only
// thing standing between a fleet-wide rename and a long tail of silent drift.
package retire

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Version names the release that retired these commands. It appears in every
// message so the reader can date the change against their own checkout.
const Version = "v1.0.0"

// Command returns a cobra command under the old name that always fails,
// naming the replacement. use is the full replacement invocation as the user
// should type it; why is one clause explaining the rule that was broken, in
// lower case and without a trailing stop.
func Command(name, use, why string) *cobra.Command {
	return &cobra.Command{
		Use:    name,
		Short:  "retired in " + Version + " - use `" + use + "`",
		Hidden: true, // keep the help output about the grammar that is current
		// The args are irrelevant: whatever follows a retired name, the answer
		// is the same. Accepting them means the error explains the rename
		// rather than cobra complaining about arity first.
		//
		// Flag parsing is off for the same reason, and it is the case that
		// actually bites: a script carrying `clog CI targets --kind bucket`
		// would otherwise die on "unknown flag: --kind", which says nothing
		// about where the command went. The old flags must be swallowed so the
		// rename is what the reader is told.
		Args:               cobra.ArbitraryArgs,
		DisableFlagParsing: true,
		SilenceUsage:       true, // the replacement is the useful output, not a usage block
		SilenceErrors:      false,
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("`%s` was retired in %s\n  why: %s\n  use: %s",
				cmd.CommandPath(), Version, why, use)
		},
	}
}

// Add registers a retired command on parent. It is the common case and keeps
// the call sites to one line each.
func Add(parent *cobra.Command, name, use, why string) {
	parent.AddCommand(Command(name, use, why))
}

// Reasons shared by more than one retirement, so the wording cannot drift
// between two commands that broke the same rule.
const (
	// BareNoun is for a noun sitting in the verb slot.
	BareNoun = "a bare noun says nothing about whether it prints a value, a list or a document"
	// WrongCardinality is for `get` used where the result is a list.
	WrongCardinality = "`get` returns one value; this returns a list, so the verb is `list`"
	// CamelCase is for names that were camelCase in a tree that is not.
	CamelCase = "camelCase reads as a Go identifier, not a command"
)
