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

// productionCmd provides BC (build-control) functionality to checkout the latest production release.
// Filters AppRelease data where Flow=="main" and Build=="prod" and checks out the latest version.
var productionCmd = &cobra.Command{
	Use:           "production",
	SilenceErrors: true,
	SilenceUsage:  true,
	Short:         "BC (build-control) Checkout latest production release",
	Long: `BC (build-control) checks out the latest production release tag.
This command:
- Loads AppRelease data from kfg configuration
- Filters for releases where Flow="main" and Build="prod"
- Finds the latest (most recent) production release
- Checks out that version tag using CheckoutTag()

Use --dryrun flag to see what git command would be executed without actually running it.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Use the global dryrun flag
		dryrun := DryRun()

		// Ensure kfg configuration and releases are loaded
		if kfg.Raw == nil {
			slog.Error("configuration not loaded. Make sure kfg.Konfigure() has been called.")
			os.Exit(1)
		}

		if len(Releases()) == 0 {
			slog.Error("no release data available. Make sure kfg.LoadReleases() has been called.")
			os.Exit(1)
		}

		version, err := GitTagProduction()
		if err != nil {
			slog.Error("Cannot get production tag", "err", err)
			os.Exit(1)
		}

		if dryrun {
			// Print the equivalent git command that would be executed
			fmt.Printf("git checkout %s\n", version)
			return
		}

		// Perform the actual checkout using CheckoutTag
		slog.Info("Checking out production release: " + version)
		err = CheckoutTag(version)
		if err != nil {
			slog.Error("Error checking out tag", "version", version, "error", err)
			os.Exit(1)
		}

		slog.Info("Successfully checked out production release %" + version)
	},
}

func init() {
	// Add production command to the checkout command
	checkoutCmd.AddCommand(productionCmd)
}
