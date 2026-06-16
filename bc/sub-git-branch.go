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

// branchCmd prints the current git branch
var branchCmd = &cobra.Command{
	Use:           "branch",
	SilenceErrors: true,
	SilenceUsage:  true,
	Short:         "Print the current git branch",
	Long: `Print the current git branch using the git SDK.

This command shows the name of the currently checked out branch.`,
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
			fmt.Println("HEAD")
			return nil
		}

		// Extract the branch name from the reference
		branchName := head.Name().Short()
		fmt.Println(branchName)
		return nil
	},
}

func init() {
	// Add branch subcommand to the git command
	gitCmd.AddCommand(branchCmd)
}
