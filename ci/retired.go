//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/

package ci

import "github.com/mrmxf/util/retire"

// Every CI spelling that v1.0.0 changed, kept reachable as an error.
//
// The namespace itself moved from `ci` to `CI`, because a Capitalised name
// means "shipped" in clog and lowercase means "yours". That one is not
// retired: ResolveCase folds `ci` to `CI` whenever nothing has claimed the
// lowercase name, so the old spelling keeps working unless a repo has
// deliberately taken it.
func init() {
	// A bare noun printed a document without saying so.
	retire.Add(Command, "policy", "clog CI show policy", retire.BareNoun)
	retire.Add(Command, "targets", "clog CI target list", retire.BareNoun)

	// `resolve` was a verb, but it named the act rather than the answer. What
	// comes back is a document about the event that triggered this run.
	retire.Add(Command, "resolve", "clog CI show event",
		"`resolve` named the work, not the result; what it prints is the event")

	// The verbs moved into the tree, so the noun no longer has to be parsed
	// out of an argument or flipped by a flag.
	retire.Add(Command, "env", "clog CI mode show",
		"`env` was renamed `mode` before v1.0.0 and is now a verb under it")

	// `get` returns one value; these return lists.
	for _, n := range []string{"tools", "chk", "make", "names"} {
		retire.Add(stackGetCmd, n, "clog CI stack list "+n, retire.WrongCardinality)
	}
}
