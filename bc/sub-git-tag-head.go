//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

package bc

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/spf13/cobra"
)

// headCmd provides BC (build-control) functionality to display the current HEAD tag.
// Uses the go-git SDK to retrieve tag information from the repository.
var headCmd = &cobra.Command{
	Use:           "head",
	SilenceErrors: true,
	SilenceUsage:  true,
	Short:         "BC (build-control) Display HEAD tag",
	Long: `BC (build-control) displays the current HEAD tag if one exists.
This command uses the go-git SDK to:
- Open the current Git repository
- Find the HEAD commit
- Look for tags that point to the HEAD commit
- Display the tag name if found

If no tag points to HEAD, it will indprint blank.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Open the Git repository in the current directory
		repo, err := git.PlainOpen(".")
		if err != nil {
			slog.Debug("Error opening repository", "error", err)
			os.Exit(1)
		}

		// Get the HEAD reference
		headRef, err := repo.Head()
		if err != nil {
			slog.Debug("Error getting HEAD", "error", err)
			os.Exit(1)
		}

		// Get all tags
		tagRefs, err := repo.Tags()
		if err != nil {
			slog.Debug("Error getting tags", "error", err)
			os.Exit(1)
		}

		// Find tags that point to the HEAD commit
		headCommit := headRef.Hash()
		var headTags []string

		err = tagRefs.ForEach(func(tagRef *plumbing.Reference) error {
			// Check if this tag points to the HEAD commit
			if tagRef.Hash() == headCommit {
				// Extract tag name from reference (remove refs/tags/ prefix)
				tagName := tagRef.Name().Short()
				headTags = append(headTags, tagName)
			}
			return nil
		})

		if err != nil {
			slog.Debug("Error iterating tags", "error", err)
			os.Exit(1)
		}

		// Display results
		if len(headTags) == 0 {
			slog.Debug("No tag found at HEAD", "commit", headCommit.String()[:8])
		} else if len(headTags) == 1 {
			fmt.Println(headTags[0])
		} else {
			fmt.Printf("HEAD tags: %v\n", headTags)
		}
	},
}

func init() {
	// Add head command to the tag command
	tagCmd.AddCommand(headCmd)
}
