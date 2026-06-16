//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

package bc

import (
	"fmt"
	"log/slog"
	"os"
	"sort"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/spf13/cobra"
)

// listCmd provides BC (build-control) functionality to list checkout children.
// Lists branches and tags in the style of git tag command.
var listCmd = &cobra.Command{
	Use:           "list",
	SilenceErrors: true,
	SilenceUsage:  true,
	Short:         "BC (build-control) List checkout children",
	Long: `BC (build-control) lists checkout children (branches and tags) in the style of git tag.
This command uses the go-git SDK to:
- Open the current Git repository
- Retrieve all local branches
- Retrieve all tags
- Display them in a simple list format, sorted alphabetically

Output format matches the style of 'git tag' command - one item per line.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Open the Git repository in the current directory
		repo, err := git.PlainOpen(".")
		if err != nil {
			slog.Error("Error opening repository", "error", err)
			os.Exit(1)
		}

		var items []string

		// Get all branches
		branchRefs, err := repo.Branches()
		if err != nil {
			slog.Error("Error getting branches", "error", err)
			os.Exit(1)
		}

		err = branchRefs.ForEach(func(branchRef *plumbing.Reference) error {
			// Extract branch name from reference (remove refs/heads/ prefix)
			branchName := branchRef.Name().Short()
			items = append(items, branchName)
			return nil
		})

		if err != nil {
			slog.Error("Error iterating branches", "error", err)
			os.Exit(1)
		}

		// Get all tags
		tagRefs, err := repo.Tags()
		if err != nil {
			slog.Error("Error getting tags", "error", err)
			os.Exit(1)
		}

		err = tagRefs.ForEach(func(tagRef *plumbing.Reference) error {
			// Extract tag name from reference (remove refs/tags/ prefix)
			tagName := tagRef.Name().Short()
			items = append(items, tagName)
			return nil
		})

		if err != nil {
			slog.Error("Error iterating tags", "error", err)
			os.Exit(1)
		}

		// Sort items alphabetically (like git tag does)
		sort.Strings(items)

		// Display results in git tag style - one per line
		for _, item := range items {
			fmt.Println(item)
		}
	},
}

func init() {
	// Add list command to the checkout command
	checkoutCmd.AddCommand(listCmd)
}
