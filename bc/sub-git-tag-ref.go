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

// GitTagRef returns the release tag at HEAD, else the nearest one before it.
func GitTagRef() string {
	g, err := buildinfo.ReadGitState(".")
	if err != nil {
		slog.Debug("cannot read git state", "err", err)
		return ""
	}
	return g.Nearest
}

// refCmd prints the release tag this checkout builds from (git, not releases.yaml)
var refCmd = &cobra.Command{
	Use:           "ref",
	SilenceErrors: true,
	SilenceUsage:  true,
	Short:         "Print the release tag at HEAD, else the nearest one before it",
	Long: `Print the release tag (vX.Y.Z) this checkout is built from: the one at HEAD,
else the newest one reachable from HEAD. It comes from git; releases.yaml is
history. For the full build version (v1.2.3+dev.3.gabc1234) use
clog BC genBuildinfo --format version.`,
	Run: func(cmd *cobra.Command, args []string) {
		version := GitTagRef()
		if version == "" {
			slog.Error("no release tag (vX.Y.Z) reachable from HEAD")
			os.Exit(1)
		}

		fmt.Println(version)
	},
}

func init() {
	// Add ref subcommand to the tag command
}
