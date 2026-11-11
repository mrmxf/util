//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

package bc

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

var productionCmd = &cobra.Command{
	Use:           "production",
	SilenceErrors: true,
	SilenceUsage:  true,
	Short:         "BC (build-control) Checkout the production release (newest vX.Y.Z tag)",
	Long: `BC (build-control) checks out the production release: the newest release tag
by date (see clog BC git tag prod). --branch origin/main only considers tags
reachable from that branch - what a scheduled production rebuild wants.

The checkout is detached (a tag is not a branch). Use --dryrun to print the git
command instead of running it.`,
	Run: func(cmd *cobra.Command, args []string) {
		version, err := GitTagProduction()
		if err != nil {
			slog.Error("Cannot get production tag", "err", err)
			os.Exit(1)
		}
		if err := safeRef(version); err != nil {
			slog.Error("refusing to check out", "err", err)
			os.Exit(1)
		}
		if DryRun() {
			fmt.Printf("git checkout %s\n", version)
			return
		}
		head, headErr := revParse("HEAD")
		tagCommit, tagErr := revParse(version + "^{commit}")
		switch {
		case tagErr != nil:
			slog.Error("production tag is not a local commit", "version", version, "error", tagErr)
			os.Exit(1)
		case headErr == nil && head == tagCommit:
			// A scheduled rebuild with no commits since the release: HEAD is
			// already the release, so a checkout could only fail, never help.
			slog.Info("Already at production release: "+version, "commit", short(head))
			return
		}
		slog.Info("Checking out production release: "+version, "from", short(head), "to", short(tagCommit))
		if err := gitNetRun("-c", "advice.detachedHead=false", "checkout", "-q", version, "--"); err != nil {
			slog.Error("Error checking out tag", "version", version, "error", err)
			os.Exit(1)
		}
	},
}

// revParse resolves a local revision to its full hash. It never touches the
// network. The error carries git's stderr.
func revParse(rev string) (string, error) {
	out, err := exec.Command("git", "rev-parse", "--verify", "--quiet", "--end-of-options", rev).Output()
	if err != nil {
		return "", fmt.Errorf("git rev-parse %s: %w", rev, err)
	}
	return strings.TrimSpace(string(out)), nil
}

// short abbreviates a hash for a log line; "" stays "".
func short(hash string) string {
	if len(hash) > 8 {
		return hash[:8]
	}
	return hash
}

func init() {
	productionCmd.Flags().StringVar(&prodBranch, "branch", "", "only tags reachable from this branch, e.g. origin/main")
	checkoutCmd.AddCommand(productionCmd)
}
