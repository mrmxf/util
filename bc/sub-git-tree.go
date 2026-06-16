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
	Long: `BC (build-control) Git tree status operations provide git working tree checks:
- Check if tree is clean
- Check if tree is ahead of remote
- Check if tree is behind remote
- Check if tree has unstaged changes

Use 'clog bc git tree <command> --help' for more information about specific tree commands.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Print help when no subcommands are provided
		cmd.Help()
	},
}

// treeCleanCmd checks if the git tree is clean
var treeCleanCmd = &cobra.Command{
	Use:           "clean",
	SilenceErrors: true,
	SilenceUsage:  true,
	Short:         "Check if git tree is clean",
	Long:          "Exits with 0 if tree is clean, otherwise exits with 1",
	Run:           treeCleanRun,
}

// treeAheadCmd checks if the git tree is ahead of remote
var treeAheadCmd = &cobra.Command{
	Use:           "ahead",
	SilenceErrors: true,
	SilenceUsage:  true,
	Short:         "Check if git tree is ahead of remote",
	Long:          "Exits with 0 if tree is NOT ahead of remote, otherwise exits with 1",
	Run:           treeAheadRun,
}

// treeBehindCmd checks if the git tree is behind remote
var treeBehindCmd = &cobra.Command{
	Use:           "behind",
	SilenceErrors: true,
	SilenceUsage:  true,
	Short:         "Check if git tree is behind remote",
	Long:          "Exits with 0 if tree is NOT behind remote, otherwise exits with 1",
	Run:           treeBehindRun,
}

// treeUnstagedCmd checks if the git tree has unstaged changes
var treeUnstagedCmd = &cobra.Command{
	Use:           "unstaged",
	SilenceErrors: true,
	SilenceUsage:  true,
	Short:         "Check if git tree has unstaged changes",
	Long:          "Exits with 0 if tree has NO unstaged changes, otherwise exits with 1",
	Run:           treeUnstagedRun,
}

func init() {
	// Add tree subcommand to the git command
	gitCmd.AddCommand(treeCmd)

	// Add subcommands to tree
	treeCmd.AddCommand(treeCleanCmd)
	treeCmd.AddCommand(treeAheadCmd)
	treeCmd.AddCommand(treeBehindCmd)
	treeCmd.AddCommand(treeUnstagedCmd)
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

	// If not ahead (0 commits), exit 0
	if commitsAhead == 0 {
		slog.Debug("Git tree is not ahead of remote")
		os.Exit(0)
	}

	slog.Debug("Git tree is ahead of remote", "commits", commitsAhead)
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

	// If not behind (0 commits), exit 0
	if commitsBehind == 0 {
		slog.Debug("Git tree is not behind remote")
		os.Exit(0)
	}

	slog.Debug("Git tree is behind remote", "commits", commitsBehind)
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

	// If no unstaged changes, exit 0
	if !hasUnstaged {
		slog.Debug("Git tree has no unstaged changes")
		os.Exit(0)
	}

	slog.Debug("Git tree has unstaged changes")
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
