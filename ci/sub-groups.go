//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/

package ci

import (
	"github.com/mrmxf/util/retire"
	"github.com/spf13/cobra"
)

// scanNsCmd is the namespace: `clog CI scan <verb>`. It has two verbs, so it
// cannot be a leaf under `show` - listing the targets a sweep visits is a
// different question from printing the sweep itself.
var scanNsCmd = &cobra.Command{
	Use:           "scan",
	Short:         "what the security sweeps examine",
	SilenceErrors: true,
	SilenceUsage:  true,
	Run:           ciHelpRun,
}

func init() {
	// documents about the run as a whole
	Command.AddCommand(showCmd)
	showCmd.AddCommand(policyCmd, resolveCmd)

	// `list` at CI level is a namespace for repo-wide lists; the per-noun
	// lists live under their noun, so the verb always follows what it acts on.
	Command.AddCommand(listCmd)

	// target: list them, or read one value from $CLOG_TARGET
	targetCmd.AddCommand(targetsCmd, targetGetCmd)

	// stack: list, get, echo
	stackCmd.AddCommand(stackListCmd, stackGetCmd, stackEchoCmd)

	// scan: the axis, and the targets an artifact sweep visits
	Command.AddCommand(scanNsCmd)
	scanNsCmd.AddCommand(scanCmd, scanListCmd)

	// Both were commands before v1.0.0 and printed values. As plain namespaces
	// they printed help and exited 0, so `for t in $(clog CI stack)` looped over
	// help text and `eval "$(clog CI scan --format env)"` set nothing.
	retire.Namespace(stackCmd, "clog CI stack list", retire.BareNoun)
	retire.Namespace(scanNsCmd, "clog CI scan show", retire.BareNoun)
}
