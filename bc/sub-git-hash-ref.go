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
}

// refHashRun gets the ref tag and prints its hash from the local repository
//
// A failure prints NOTHING to stdout. Before v1.0.0 each error path printed
// the literal string "not found" there, so `h=$(clog BC git hash get prod)`
// captured that text and any caller testing -z on it saw a value. The error
// still goes to stderr via slog and the exit code is still 1; stdout is
// reserved for the hash, or for nothing at all.
func refHashRun(cmd *cobra.Command, args []string) {
	// Get the ref tag using the helper function
	refTag := GitTagRef()
	if refTag == "" {
		slog.Error("No release data available")
		os.Exit(1)
	}

	// Get the hash of the local tag
	// Use git rev-parse to get the hash of the tag
	gitCmd := exec.Command("git", "rev-parse", refTag)
	output, err := gitCmd.Output()
	if err != nil {
		slog.Error("Failed to get hash for tag", "tag", refTag, "error", err)
		os.Exit(1)
	}

	// Parse and print the hash
	hash := strings.TrimSpace(string(output))
	if hash == "" {
		slog.Error("Tag not found in local repository", "tag", refTag)
		os.Exit(1)
	}

	fmt.Println(hash)
}
