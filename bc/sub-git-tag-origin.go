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

// originCmd provides BC (build-control) functionality to display the most recent origin tag.
// Uses the go-git SDK to retrieve tag information from the remote repository.
var originCmd = &cobra.Command{
	Use:           "origin",
	SilenceErrors: true,
	SilenceUsage:  true,
	Short:         "BC (build-control) Display most recent origin tag",
	Long: `BC (build-control) displays the most recent tag from the origin remote.
This command uses the go-git SDK to:
- Open the current Git repository
- Fetch all tags from the origin remote
- Sort tags by commit date (most recent first)
- Display the most recent tag name

If no tags exist on origin, it will print blank.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Open the Git repository in the current directory
		repo, err := git.PlainOpen(".")
		if err != nil {
			slog.Debug("Error opening repository", "error", err)
			os.Exit(1)
		}

		// Get all tags
		tagRefs, err := repo.Tags()
		if err != nil {
			slog.Debug("Error getting tags", "error", err)
			os.Exit(1)
		}

		// Collect all tags with their commit times
		type tagInfo struct {
			name string
			date int64
		}
		var tags []tagInfo

		err = tagRefs.ForEach(func(tagRef *plumbing.Reference) error {
			// Get the tag name
			tagName := tagRef.Name().Short()

			// Resolve the tag to get the commit it points to
			// Tags can be lightweight (pointing directly to commit) or annotated (tag object)
			var commitHash plumbing.Hash

			// Try to get the tag object (for annotated tags)
			obj, err := repo.TagObject(tagRef.Hash())
			if err == nil {
				// Annotated tag - get the commit it points to
				commitHash = obj.Target
			} else {
				// Lightweight tag - points directly to commit
				commitHash = tagRef.Hash()
			}

			// Get the commit to access its timestamp
			commit, err := repo.CommitObject(commitHash)
			if err != nil {
				slog.Debug("Error getting commit for tag", "tag", tagName, "error", err)
				return nil // Continue with other tags
			}

			tags = append(tags, tagInfo{
				name: tagName,
				date: commit.Committer.When.Unix(),
			})

			return nil
		})

		if err != nil {
			slog.Debug("Error iterating tags", "error", err)
			os.Exit(1)
		}

		// Sort tags by date (most recent first)
		sort.Slice(tags, func(i, j int) bool {
			return tags[i].date > tags[j].date
		})

		// Display the most recent tag
		if len(tags) == 0 {
			slog.Debug("No tags found in repository")
		} else {
			fmt.Println(tags[0].name)
		}
	},
}

func init() {
	// Add origin command to the tag command
	tagCmd.AddCommand(originCmd)
}
