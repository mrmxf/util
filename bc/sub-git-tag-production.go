//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

package bc

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/mrmxf/util/buildinfo"
	"github.com/spf13/cobra"
)

// prodBranch narrows "production" to release tags reachable from a branch
// (--branch origin/main: what a scheduled rebuild should build).
var prodBranch string

// GitTagProduction returns the production release: the newest release tag
// (vX.Y.Z, by tag date) in git. releases.yaml is history, not consulted.
func GitTagProduction() (string, error) {
	return buildinfo.LatestReleaseTag(".", prodBranch)
}

var prodTagCmd = &cobra.Command{
	Use:           "prod",
	Aliases:       []string{"production"},
	SilenceErrors: true,
	SilenceUsage:  true,
	Short:         "Print the production release tag (newest vX.Y.Z tag)",
	Long: `Print the production release: the newest release tag by date.

A release tag is vX.Y.Z or X.Y.Z with nothing after it, so v1.2.0-rc1 and
v3.5.0-dev never count. --branch origin/main only considers tags reachable from
that branch. releases.yaml is not consulted: it is history.`,
	Run: func(cmd *cobra.Command, args []string) {
		version, err := GitTagProduction()
		if err != nil {
			slog.Error("Cannot get production tag", "err", err)
			os.Exit(1)
		}
		fmt.Println(version)
	},
}

func init() {
	prodTagCmd.Flags().StringVar(&prodBranch, "branch", "", "only tags reachable from this branch, e.g. origin/main")
}
