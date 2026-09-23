//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

package bc

import (
	"fmt"
	"os/exec"
	"strings"

	slog "github.com/mrmxf/util/slogger"

	"github.com/spf13/cobra"
)

// hashOriginCmd prints the hash of the origin HEAD
var hashOriginCmd = &cobra.Command{
	Use:           "origin",
	SilenceErrors: true,
	SilenceUsage:  true,
	Short:         "Print the hash of the origin HEAD",
	Long:          "Prints the Git commit hash of the origin/HEAD reference from the remote repository",
	Example:       "clog BC git hash origin",
	Run:           hashOriginRun,
}

func init() {
	// Add origin subcommand to the hash command
}

// hashOriginRun prints the hash of the origin HEAD.
//
// A CI checkout has no origin/HEAD (actions/checkout fetches one ref and often
// leaves HEAD detached), so try in order: origin/HEAD, origin/<branch>, then
// the remote itself. Nothing found is not an error - the hash is printed for
// information in a pre-build status block - so it logs at debug and prints
// nothing, rather than an ERR line in every job log.
func hashOriginRun(cmd *cobra.Command, args []string) {
	for _, ref := range originRefs() {
		if out, err := exec.Command("git", "rev-parse", "--verify", "--quiet", ref).Output(); err == nil {
			if hash := strings.TrimSpace(string(out)); hash != "" {
				fmt.Println(hash)
				return
			}
		}
	}
	if out, err := exec.Command("git", "ls-remote", "origin", "HEAD").Output(); err == nil {
		if fields := strings.Fields(string(out)); len(fields) > 0 {
			fmt.Println(fields[0])
			return
		}
	}
	slog.Debug("no origin HEAD hash: no origin/HEAD ref, no origin/<branch>, and the remote did not answer")
}

// originRefs lists the local refs that may hold the origin HEAD hash.
func originRefs() []string {
	refs := []string{"origin/HEAD"}
	if out, err := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").Output(); err == nil {
		if branch := strings.TrimSpace(string(out)); branch != "" && branch != "HEAD" {
			refs = append(refs, "origin/"+branch)
		}
	}
	return refs
}
