//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

package bc

import (
	"fmt"
	"log/slog"

	"github.com/go-git/go-git/v5"
	"github.com/spf13/cobra"
)

// suffixCmd prints the current git branch unless it's "main"
var suffixCmd = &cobra.Command{
	Use:           "suffix",
	SilenceErrors: true,
	SilenceUsage:  true,
	Short:         "Print the current git branch as a suffix (empty if main)",
	Long: `Print the current git branch name to be used as a suffix.
If the current branch is "main", prints an empty string.
Otherwise, prints the branch name.

This is commonly used in build systems to create branch-specific artifacts.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Open the repository in the current directory
		repo, err := git.PlainOpen(".")
		if err != nil {
			slog.Error("failed to open local repository", "error", err)
			return err
		}

		// Get the current HEAD reference
		head, err := repo.Head()
		if err != nil {
			slog.Error("failed to get HEAD reference", "error", err)
			return err
		}

		// Check if we're on a branch (not detached HEAD)
		if !head.Name().IsBranch() {
			// For detached HEAD, return empty string
			return nil
		}

		// Extract the branch name from the reference
		branchName := head.Name().Short()

		// If branch is "main", return empty string, otherwise return the branch name
		if branchName == "main" {
			// Print nothing (empty string)
			return nil
		}

		fmt.Println(branchName)
		return nil
	},
}

func init() {
	// Add suffix subcommand to the git command
	gitCmd.AddCommand(suffixCmd)
}
