//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

package bc

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	slog "github.com/mrmxf/util/slogger"

	"github.com/spf13/cobra"
)

// refHashCmd prints the hash of the ref tag from the local repository
var refHashCmd = &cobra.Command{
	Use:           "ref",
	SilenceErrors: true,
	SilenceUsage:  true,
	Short:         "Print the hash of the ref tag",
	Long:          "Prints the hash of the local ref tag from releases.yaml",
	Run:           refHashRun,
}

func init() {
	// Add ref subcommand to the hash command
	hashCmd.AddCommand(refHashCmd)
}

// refHashRun gets the ref tag and prints its hash from the local repository
func refHashRun(cmd *cobra.Command, args []string) {
	// Get the ref tag using the helper function
	refTag := GitTagRef()
	if refTag == "" {
		fmt.Println("not found")
		slog.Error("No release data available")
		os.Exit(1)
	}

	// Get the hash of the local tag
	// Use git rev-parse to get the hash of the tag
	gitCmd := exec.Command("git", "rev-parse", refTag)
	output, err := gitCmd.Output()
	if err != nil {
		fmt.Println("not found")
		slog.Error("Failed to get hash for tag", "tag", refTag, "error", err)
		os.Exit(1)
	}

	// Parse and print the hash
	hash := strings.TrimSpace(string(output))
	if hash == "" {
		fmt.Println("not found")
		slog.Error("Tag not found in local repository", "tag", refTag)
		os.Exit(1)
	}

	fmt.Println(hash)
}
