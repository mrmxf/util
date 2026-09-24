//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

package bc

import (
	"os"
	"os/exec"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	slog "github.com/mrmxf/util/slogger"
	"github.com/spf13/cobra"
)

// treeCmd provides BC (build-control) Git tree status operations
var treeCmd = &cobra.Command{
	Use:           "tree",
	SilenceErrors: true,
	SilenceUsage:  true,
	Short:         "BC (build-control) Git tree status operations",
	Long: `BC (build-control) Git tree status operations ask a question about the
working tree and answer with an exit code only - nothing is printed.

  clog BC git tree is clean       0 if the tree is clean
  clog BC git tree is ahead       0 if the tree is AHEAD of origin
  clog BC git tree is behind      0 if the tree is BEHIND origin
  clog BC git tree has unstaged   0 if there ARE unstaged changes

Every one of these is true when it exits 0, so a shell reads the way it looks:

  if clog BC git tree is ahead; then git push; fi
  clog BC git tree is clean || exit 1

Before v1.0.0 ` + "`ahead`" + `, ` + "`behind`" + ` and ` + "`unstaged`" + ` were inverted - they exited 0
when the tree was NOT in that state - so the line above did the opposite of
what it said. The old spellings are retired rather than fixed in place,
because a predicate that changes meaning while keeping its name is the one
rename that cannot fail loudly.`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help() //nolint:errcheck // help output is best-effort
	},
}

// treeIsCmd groups the state predicates: `clog BC git tree is <state>`.
var treeIsCmd = &cobra.Command{
	Use:           "is",
	SilenceErrors: true,
	SilenceUsage:  true,
	Short:         "is the tree clean, ahead or behind? (exit code only)",
	Args:          cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help() //nolint:errcheck // help output is best-effort
	},
}

// treeHasCmd groups the possession predicates: `clog BC git tree has <thing>`.
var treeHasCmd = &cobra.Command{
	Use:           "has",
	SilenceErrors: true,
	SilenceUsage:  true,
	Short:         "does the tree have unstaged changes? (exit code only)",
	Args:          cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help() //nolint:errcheck // help output is best-effort
	},
}

// treeCleanCmd - `clog BC git tree is clean`
var treeCleanCmd = &cobra.Command{
	Use:           "clean",
	SilenceErrors: true,
	SilenceUsage:  true,
	Short:         "exit 0 if the working tree is clean",
	Long:          "Exits 0 if the tree IS clean, 1 if it is not.",
	Run:           treeCleanRun,
}

// treeAheadCmd - `clog BC git tree is ahead`
var treeAheadCmd = &cobra.Command{
	Use:           "ahead",
	SilenceErrors: true,
	SilenceUsage:  true,
	Short:         "exit 0 if the working tree is ahead of origin",
	Long:          "Exits 0 if the tree IS ahead of origin, 1 if it is not.",
	Run:           treeAheadRun,
}

// treeBehindCmd - `clog BC git tree is behind`
var treeBehindCmd = &cobra.Command{
	Use:           "behind",
	SilenceErrors: true,
	SilenceUsage:  true,
	Short:         "exit 0 if the working tree is behind origin",
	Long:          "Exits 0 if the tree IS behind origin, 1 if it is not.",
	Run:           treeBehindRun,
}

// treeUnstagedCmd - `clog BC git tree has unstaged`
var treeUnstagedCmd = &cobra.Command{
	Use:           "unstaged",
	SilenceErrors: true,
	SilenceUsage:  true,
	Short:         "exit 0 if the working tree has unstaged changes",
	Long:          "Exits 0 if there ARE unstaged changes, 1 if there are none.",
	Run:           treeUnstagedRun,
}

func init() {
	gitCmd.AddCommand(treeCmd)

	treeCmd.AddCommand(treeIsCmd)
	treeCmd.AddCommand(treeHasCmd)

	treeIsCmd.AddCommand(treeCleanCmd)
	treeIsCmd.AddCommand(treeAheadCmd)
	treeIsCmd.AddCommand(treeBehindCmd)
	treeHasCmd.AddCommand(treeUnstagedCmd)
}

// treeCleanRun checks if the git tree is clean using native git command.
//
// Note: This function uses native 'git status --porcelain' instead of go-git API.
// While go-git is used elsewhere in this file, it has known issues in WSL environments
// where it reports false positives for modified files due to filesystem metadata
// mismatches (e.g., file timestamps, permission bits) that native git correctly ignores.
// Native git properly handles index refreshing and respects .gitattributes, core.autocrlf,
// and core.filemode settings, making it more reliable for this specific check.
func treeCleanRun(cmd *cobra.Command, args []string) {
	// Use native git status --porcelain which returns empty output if clean
	gitCmd := exec.Command("git", "status", "--porcelain")
	output, err := gitCmd.Output()
	if err != nil {
		slog.Error("Error getting git status", "error", err)
		os.Exit(1)
	}

	// If output is empty, tree is clean (ignoring untracked files is handled by --porcelain)
	if len(output) == 0 {
		slog.Debug("Git tree is clean")
		os.Exit(0)
	}

	slog.Debug("Git tree is not clean")
	os.Exit(1)
}

// treeAheadRun checks if the git tree is ahead of remote using go-git API
func treeAheadRun(cmd *cobra.Command, args []string) {
	// Open the Git repository in the current directory
	repo, err := git.PlainOpen(".")
	if err != nil {
		slog.Error("Error opening repository", "error", err)
		os.Exit(1)
	}

	// Get the HEAD reference
	headRef, err := repo.Head()
	if err != nil {
		slog.Error("Error getting HEAD", "error", err)
		os.Exit(1)
	}

	// Get the remote tracking branch
	remoteName := "origin"
	branchName := headRef.Name().Short()
	remoteRefName := plumbing.NewRemoteReferenceName(remoteName, branchName)

	remoteRef, err := repo.Reference(remoteRefName, true)
	if err != nil {
		slog.Error("Error getting remote reference", "error", err)
		os.Exit(1)
	}

	// Count commits ahead
	commitsAhead, err := countCommitsBetween(repo, remoteRef.Hash(), headRef.Hash())
	if err != nil {
		slog.Error("Error counting commits ahead", "error", err)
		os.Exit(1)
	}

	// The predicate is "is the tree ahead", so being ahead is the 0 case.
	if commitsAhead > 0 {
		slog.Debug("Git tree is ahead of remote", "commits", commitsAhead)
		os.Exit(0)
	}

	slog.Debug("Git tree is not ahead of remote")
	os.Exit(1)
}

// treeBehindRun checks if the git tree is behind remote using go-git API
func treeBehindRun(cmd *cobra.Command, args []string) {
	// Open the Git repository in the current directory
	repo, err := git.PlainOpen(".")
	if err != nil {
		slog.Error("Error opening repository", "error", err)
		os.Exit(1)
	}

	// Get the HEAD reference
	headRef, err := repo.Head()
	if err != nil {
		slog.Error("Error getting HEAD", "error", err)
		os.Exit(1)
	}

	// Get the remote tracking branch
	remoteName := "origin"
	branchName := headRef.Name().Short()
	remoteRefName := plumbing.NewRemoteReferenceName(remoteName, branchName)

	remoteRef, err := repo.Reference(remoteRefName, true)
	if err != nil {
		slog.Error("Error getting remote reference", "error", err)
		os.Exit(1)
	}

	// Count commits behind
	commitsBehind, err := countCommitsBetween(repo, headRef.Hash(), remoteRef.Hash())
	if err != nil {
		slog.Error("Error counting commits behind", "error", err)
		os.Exit(1)
	}

	// The predicate is "is the tree behind", so being behind is the 0 case.
	if commitsBehind > 0 {
		slog.Debug("Git tree is behind remote", "commits", commitsBehind)
		os.Exit(0)
	}

	slog.Debug("Git tree is not behind remote")
	os.Exit(1)
}

// treeUnstagedRun checks if the git tree has unstaged changes using go-git API
func treeUnstagedRun(cmd *cobra.Command, args []string) {
	// Open the Git repository in the current directory
	repo, err := git.PlainOpen(".")
	if err != nil {
		slog.Error("Error opening repository", "error", err)
		os.Exit(1)
	}

	// Get the working tree
	worktree, err := repo.Worktree()
	if err != nil {
		slog.Error("Error getting worktree", "error", err)
		os.Exit(1)
	}

	// Get the status
	status, err := worktree.Status()
	if err != nil {
		slog.Error("Error getting status", "error", err)
		os.Exit(1)
	}

	// Check for unstaged changes (modified or deleted files not staged)
	// Unstaged changes are when Worktree has modifications that are NOT staged
	hasUnstaged := false
	for _, fileStatus := range status {
		// Skip untracked files (both staging and worktree are Untracked)
		if fileStatus.Staging == git.Untracked && fileStatus.Worktree == git.Untracked {
			continue
		}

		// Check if worktree has modifications that differ from staging
		// Modified, Deleted, Renamed, Copied in worktree = unstaged changes
		if fileStatus.Worktree == git.Modified ||
			fileStatus.Worktree == git.Deleted ||
			fileStatus.Worktree == git.Renamed ||
			fileStatus.Worktree == git.Copied {
			hasUnstaged = true
			break
		}
	}

	// The predicate is "has unstaged changes", so having them is the 0 case.
	if hasUnstaged {
		slog.Debug("Git tree has unstaged changes")
		os.Exit(0)
	}

	slog.Debug("Git tree has no unstaged changes")
	os.Exit(1)
}

// countCommitsBetween counts the number of commits between two hashes
// Returns the number of commits that are in 'to' but not in 'from'
func countCommitsBetween(repo *git.Repository, from, to plumbing.Hash) (int, error) {
	// If hashes are the same, no commits between them
	if from == to {
		return 0, nil
	}

	// Get commit iterator starting from 'to'
	commitIter, err := repo.Log(&git.LogOptions{From: to})
	if err != nil {
		return 0, err
	}
	defer commitIter.Close()

	// Count commits until we reach 'from'
	count := 0
	err = commitIter.ForEach(func(c *object.Commit) error {
		if c.Hash == from {
			// Found the 'from' commit, stop counting
			return nil
		}
		count++
		return nil
	})

	if err != nil {
		return 0, err
	}

	return count, nil
}
