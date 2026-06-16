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
	Example:       "clog bc git hash origin",
	Run:           hashOriginRun,
}

func init() {
	// Add origin subcommand to the hash command
	hashCmd.AddCommand(hashOriginCmd)
}

// hashOriginRun prints the hash of the origin HEAD
func hashOriginRun(cmd *cobra.Command, args []string) {
	// Execute git rev-parse origin/HEAD to get the commit hash
	gitCmd := exec.Command("git", "rev-parse", "origin/HEAD")
	output, err := gitCmd.Output()
	if err != nil {
		slog.Error("failed to get origin HEAD hash", "error", err)
		return
	}

	// Trim whitespace and print the hash
	hash := strings.TrimSpace(string(output))
	fmt.Println(hash)
}
