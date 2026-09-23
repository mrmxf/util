//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/

package bc

import "github.com/mrmxf/util/retire"

// Every BC spelling that v1.0.0 changed, kept reachable as an error.
//
// These are not aliases. A retired command never does the old thing - it
// prints the replacement and exits 1. Two of them could not have been aliased
// even in principle, because the behaviour changed underneath the name:
// `git tree ahead|behind|unstaged` and `stash has` were all inverted, so an
// alias would have silently returned the opposite answer to every existing
// caller. Failing is the only honest option for those, and once they had to
// fail the rest were made consistent with them.
func init() {
	// ── nouns that sat in the verb slot ──────────────────────────────────
	retire.Add(gitCmd, "branch", "clog BC git get branch", retire.BareNoun)
	retire.Add(gitCmd, "suffix", "clog BC git get suffix", retire.BareNoun)

	for _, n := range []string{"head", "origin", "prod", "ref"} {
		retire.Add(tagCmd, n, "clog BC git tag get "+n, retire.BareNoun)
		retire.Add(hashCmd, n, "clog BC git hash get "+n, retire.BareNoun)
	}
	retire.Add(tagCmd, "production", "clog BC git tag get prod", retire.BareNoun)

	for _, n := range []string{"version", "date", "flow", "note", "build"} {
		retire.Add(releasesCmd, n, "clog BC releases get "+n, retire.BareNoun)
	}
	// `yaml` named the file format, not the thing being asked for.
	retire.Add(releasesCmd, "yaml", "clog BC releases get path",
		"the value is a path; `yaml` named its format, which the caller already knew")

	retire.Add(Command, "linkerpath", "clog BC get linkerpath", retire.BareNoun)

	// ── inverted predicates: the dangerous ones ──────────────────────────
	const flipped = "this exited 0 when the answer was NO, so `if` ran the wrong branch; " +
		"the replacement exits 0 when the answer is YES"
	retire.Add(treeCmd, "clean", "clog BC git tree is clean", retire.BareNoun)
	retire.Add(treeCmd, "ahead", "clog BC git tree is ahead", flipped)
	retire.Add(treeCmd, "behind", "clog BC git tree is behind", flipped)
	retire.Add(treeCmd, "unstaged", "clog BC git tree has unstaged", flipped)
	retire.Add(stashCmd, "hasError", "clog BC stash has error", retire.CamelCase)

	// ── camelCase ────────────────────────────────────────────────────────
	retire.Add(Command, "stashLog", "clog BC stash log", retire.CamelCase)
	retire.Add(Command, "genBuildinfo", "clog BC gen buildinfo", retire.CamelCase)
	retire.Add(Command, "genbuildinfo", "clog BC gen buildinfo", retire.CamelCase)
	retire.Add(Command, "buildinfo", "clog BC gen buildinfo",
		"a bare `buildinfo` read as a printer; it generates linker flags")

	// ── a predicate that looked like a printer ───────────────────────────
	retire.Add(Command, "is", "clog BC releases is <field> <value>",
		"`is` compares a field of the newest release, so it belongs under `releases`")
}
