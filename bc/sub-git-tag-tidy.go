//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

package bc

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/spf13/cobra"
)

// tidyCmd checks and manages git tags based on the current release version
var tidyCmd = &cobra.Command{
	Use:           "tidy",
	SilenceErrors: true,
	SilenceUsage:  true,
	Short:         "Check and manage git tags for the current release",
	Long: `Check local and remote git repositories for the tag returned by GitTagRef().
If the tag exists, report an error and exit with status 1.
If the --force/-F flag is set, delete existing tags, create new tag locally, and push it.
Use --dryrun to see what operations would be performed without executing them.`,
	Run: func(cmd *cobra.Command, args []string) {
		tag := GitTagRef()
		if tag == "" {
			slog.Error("tidy command: no release data available")
			os.Exit(1)
		}

		force, _ := cmd.Flags().GetBool("force")
		dryrun := DryRun()

		// Check if tag exists locally
		localExists := checkLocalTagExists(tag)
		remoteExists := checkRemoteTagExists(tag)

		if dryrun {
			fmt.Printf("Dry run mode - would process tag: %s\n", tag)
			fmt.Printf("  Local tag exists: %t\n", localExists)
			fmt.Printf("  Remote tag exists: %t\n", remoteExists)
			fmt.Printf("  Force flag: %t\n", force)

			if localExists || remoteExists {
				if !force {
					fmt.Printf("  Would exit with error: tag already exists\n")
					return
				}
				fmt.Printf("  Would delete existing tags:\n")
				if localExists {
					fmt.Printf("    - git tag -d %s\n", tag)
				}
				if remoteExists {
					fmt.Printf("    - git push origin :refs/tags/%s\n", tag)
				}
			}

			fmt.Printf("  Would create and push tag:\n")
			fmt.Printf("    - git tag %s\n", tag)
			fmt.Printf("    - git push origin %s\n", tag)
			return
		}

		if localExists || remoteExists {
			if !force {
				slog.Error("tag already exists", "tag", tag, "local", localExists, "remote", remoteExists)
				os.Exit(1)
			}

			// Force flag is set, delete existing tags
			if localExists {
				if err := deleteLocalTag(tag); err != nil {
					slog.Error("failed to delete local tag", "tag", tag, "error", err)
					os.Exit(1)
				}
				slog.Debug("deleted local tag", "tag", tag)
			}

			if remoteExists {
				if err := deleteRemoteTag(tag); err != nil {
					slog.Error("failed to delete remote tag", "tag", tag, "error", err)
					os.Exit(1)
				}
				slog.Debug("deleted remote tag", "tag", tag)
			}
		}

		// Create and push the tag
		hash, err := createAndPushTag(tag)
		if err != nil {
			slog.Error("failed to create and push tag", "tag", tag, "error", err)
			os.Exit(1)
		}

		slog.Info("successfully created and pushed tag", "tag", tag, "hash", hash)
		fmt.Printf("Successfully created and pushed tag %s (hash: %s)\n", tag, hash[:8])
	},
}

// checkLocalTagExists checks if a tag exists in the local repository
func checkLocalTagExists(tag string) bool {
	repo, err := git.PlainOpen(".")
	if err != nil {
		return false
	}

	tagRef := plumbing.NewTagReferenceName(tag)
	_, err = repo.Reference(tagRef, true)
	return err == nil
}

// checkRemoteTagExists checks if a tag exists in the remote repository
func checkRemoteTagExists(tag string) bool {
	if err := safeRef(tag); err != nil {
		slog.Error("refusing remote tag lookup", "error", err)
		return false
	}
	output, err := gitNet("ls-remote", "--tags", "origin", "refs/tags/"+tag)
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(output)) != ""
}

// deleteLocalTag deletes a local tag
func deleteLocalTag(tag string) error {
	if err := safeRef(tag); err != nil {
		return err
	}
	// `--` ends option parsing so a tag can never be read as a git flag (S1).
	return exec.Command("git", "tag", "-d", "--", tag).Run()
}

// deleteRemoteTag deletes a remote tag
func deleteRemoteTag(tag string) error {
	if err := safeRef(tag); err != nil {
		return err
	}
	return gitNetRun("push", "origin", ":refs/tags/"+tag)
}

// createAndPushTag creates a local tag and pushes it to remote, returning the commit hash
func createAndPushTag(tag string) (string, error) {
	if err := safeRef(tag); err != nil {
		return "", err
	}

	repo, err := git.PlainOpen(".")
	if err != nil {
		slog.Error("failed to open repository", "error", err)
		return "", err
	}

	// Get HEAD commit
	head, err := repo.Head()
	if err != nil {
		slog.Error("failed to get HEAD", "error", err)
		return "", err
	}

	hash := head.Hash()

	// Create the tag locally using git command (for better compatibility).
	// `--` ends option parsing so the tag can never be read as a git flag (S1).
	if err := exec.Command("git", "tag", "--", tag).Run(); err != nil {
		slog.Error("failed to create local tag", "error", err)
		return "", err
	}

	// Push the tag to remote (timeout-bounded, R4)
	if err := gitNetRun("push", "origin", "refs/tags/"+tag); err != nil {
		slog.Error("failed to push tag", "error", err)
		return "", err
	}

	return hash.String(), nil
}

func init() {
	// Add tidy subcommand to the tag command
	tagCmd.AddCommand(tidyCmd)

	// Add --force/-F flag
	tidyCmd.Flags().BoolP("force", "F", false, "Force delete existing tags before creating new one")
}
