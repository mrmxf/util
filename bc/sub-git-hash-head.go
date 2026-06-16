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

// hashHeadCmd prints the hash of the local HEAD
var hashHeadCmd = &cobra.Command{
	Use:           "head",
	SilenceErrors: true,
	SilenceUsage:  true,
	Short:         "Print the hash of the local HEAD",
	Long:          "Prints the Git commit hash of the current HEAD reference",
	Example:       "clog bc git hash head",
	Run:           hashHeadRun,
}

func init() {
	// Add head subcommand to the hash command
	hashCmd.AddCommand(hashHeadCmd)
}

// hashHeadRun prints the hash of the local HEAD
func hashHeadRun(cmd *cobra.Command, args []string) {
	// Execute git rev-parse HEAD to get the commit hash
	gitCmd := exec.Command("git", "rev-parse", "HEAD")
	output, err := gitCmd.Output()
	if err != nil {
		slog.Error("failed to get HEAD hash", "error", err)
		return
	}

	// Trim whitespace and print the hash
	hash := strings.TrimSpace(string(output))
	fmt.Println(hash)
}
