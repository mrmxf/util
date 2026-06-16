//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

package bc

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/mrmxf/util/kfg"
	"github.com/spf13/cobra"
)

// GitTagProduction returns the current version reference from releases
// Note that it is the responsibility of the manager of releases.yaml to add or remove a 'v' prefix
func GitTagProduction() (string, error) {
	noProdTag := fmt.Errorf("no {flow:\"main\", build:\"prod\"} release in %s", ReleasesPath())

	if len(Releases()) == 0 {
		return "", noProdTag
	}

	// Filter releases where Flow == "main" and Build == "prod"
	slog.Debug(fmt.Sprintf("%d lines of release information", len(Releases())))
	var prodReleases []kfg.AppRelease
	for _, release := range Releases() {
		if release.Flow == "main" && release.Build == "prod" {
			prodReleases = append(prodReleases, release)
			slog.Debug(fmt.Sprintf("  ✅ %10s flow:%7s build:%s", release.Version, release.Flow, release.Build))
		} else {
			slog.Debug(fmt.Sprintf("  ✖️ %10s flow:%7s build:%s", release.Version, release.Flow, release.Build))
		}
	}

	if len(prodReleases) == 0 {
		return "", noProdTag
	}

	return prodReleases[0].Version, nil
}

// refCmd prints the current version number in releases.yaml
var prodTagCmd = &cobra.Command{
	Use:           "prod",
	SilenceErrors: true,
	SilenceUsage:  true,
	Short:         "Print the production tag ref",
	Long:          `Print the production tag ref from releases.yaml`,
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
	// Add ref subcommand to the tag command
	tagCmd.AddCommand(prodTagCmd)
}
