//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

package bc

import (
	"fmt"
	"os"
	"strings"

	slog "github.com/mrmxf/util/slogger"

	"github.com/spf13/cobra"
)

// productionHashCmd prints the hash of the production tag from the remote
var productionHashCmd = &cobra.Command{
	Use:           "prod",
	SilenceErrors: true,
	SilenceUsage:  true,
	Short:         "Print the hash of the production tag",
	Long:          "Prints the hash of the remote production tag from releases.yaml",
	Run:           productionHashRun,
}

func init() {
	// Add prod subcommand to the hash command
	hashCmd.AddCommand(productionHashCmd)
}

// productionHashRun gets the production tag and prints its hash from the remote
func productionHashRun(cmd *cobra.Command, args []string) {
	// Get the production tag using the helper function
	prodTag, err := GitTagProduction()
	if err != nil {
		fmt.Println("not found")
		slog.Error("Cannot get production tag", "err", err)
		os.Exit(1)
	}

	// Get the hash of the remote tag (timeout-bounded, R4)
	output, err := gitNet("ls-remote", "origin", fmt.Sprintf("refs/tags/%s", prodTag))
	if err != nil {
		fmt.Println("not found")
		slog.Error("Failed to get hash for tag", "tag", prodTag, "error", err)
		os.Exit(1)
	}

	// Parse the output (format: "<hash>\trefs/tags/<tag>")
	outputStr := strings.TrimSpace(string(output))
	if outputStr == "" {
		fmt.Println("not found")
		slog.Error("Tag not found in remote", "tag", prodTag)
		os.Exit(1)
	}

	// Extract the hash (first field before tab)
	fields := strings.Fields(outputStr)
	if len(fields) < 1 {
		fmt.Println("not found")
		slog.Error("Invalid output from git ls-remote", "output", outputStr)
		os.Exit(1)
	}

	hash := fields[0]
	fmt.Println(hash)
}
