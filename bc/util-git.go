//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

package bc

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

// Checkout performs a generic git checkout operation to the specified reference.
// It accepts any valid git reference (branch, tag, commit SHA).
func Checkout(ref string) error {
	repo, err := git.PlainOpen(".")
	if err != nil {
		slog.Error("failed to open local repository", "error", err)
		return err
	}

	worktree, err := repo.Worktree()
	if err != nil {
		slog.Error("failed to get worktree", "error", err)
		return err
	}

	// Try to resolve as a hash first
	if plumbing.IsHash(ref) {
		hash := plumbing.NewHash(ref)
		err = worktree.Checkout(&git.CheckoutOptions{
			Hash: hash,
		})
		if err != nil {
			slog.Error("failed to checkout", "ref", ref, "error", err)
			return err
		}
		return nil
	}

	// Parse the reference
	refName := plumbing.ReferenceName(ref)

	// Try to resolve the reference
	resolvedRef, err := repo.Reference(refName, true)
	if err != nil {
		slog.Error("failed to resolve reference", "ref", ref, "error", err)
		return err
	}

	// Get the hash
	hash := resolvedRef.Hash()

	// Try to get the object to see what type it is
	obj, err := repo.Object(plumbing.AnyObject, hash)
	if err != nil {
		slog.Error("failed to get object", "ref", ref, "error", err)
		return err
	}

	// If it's a tag object (annotated tag), get the target commit
	if obj.Type() == plumbing.TagObject {
		tagObj, err := repo.TagObject(hash)
		if err != nil {
			slog.Error("failed to get tag object", "ref", ref, "error", err)
			return err
		}
		hash = tagObj.Target
	}

	// Perform the checkout with the final hash
	err = worktree.Checkout(&git.CheckoutOptions{
		Hash: hash,
	})
	if err != nil {
		// Check if the checkout actually succeeded despite the error
		// This can happen when there are unstaged changes but the checkout still worked
		headRef, headErr := repo.Head()
		if headErr == nil && headRef.Hash() == hash {
			// Checkout succeeded, downgrade error to warning
			slog.Warn("checkout succeeded but go-git reported error", "error", err)
			return nil
		}
		slog.Error("failed to checkout", "ref", ref, "error", err)
		return err
	}

	return nil
}

// CheckoutTag performs a git checkout to a specific tag with validation.
// It validates that the string is a tag, exists locally and remotely, and has the same SHA.
func CheckoutTag(tag string) error {
	repo, err := git.PlainOpen(".")
	if err != nil {
		slog.Error("failed to open repository", "error", err)
		return err
	}

	// Validate that the reference is a tag
	tagRef := plumbing.NewTagReferenceName(tag)
	localTagRef, err := repo.Reference(tagRef, true)
	if err != nil {
		slog.Error("tag does not exist locally", "tag", tag, "error", err)
		return err
	}

	// Get the local tag SHA
	localTagSHA := localTagRef.Hash()

	// Check if tag exists remotely and get its SHA
	remoteTagSHA, err := getRemoteTagSHA(tag)
	if err != nil {
		slog.Error("failed to get remote tag SHA", "tag", tag, "error", err)
		return err
	}

	// Validate that local and remote SHAs match
	if localTagSHA.String() != remoteTagSHA {
		slog.Error("tag SHA mismatch", "tag", tag, "local_sha", localTagSHA.String()[:8], "remote_sha", remoteTagSHA[:8])
		return fmt.Errorf("tag %s: local SHA (%s) does not match remote SHA (%s)",
			tag, localTagSHA.String()[:8], remoteTagSHA[:8])
	}

	// Perform the checkout using the generic Checkout function
	return Checkout(tagRef.String())
}

// CheckoutBranch performs a git checkout to a specific branch.
func CheckoutBranch(branch string) error {
	branchRef := plumbing.NewBranchReferenceName(branch)
	return Checkout(branchRef.String())
}

// CheckoutRef performs a git checkout to any reference (branch, tag, or commit).
func CheckoutRef(ref string) error {
	return Checkout(ref)
}

// getRemoteTagSHA fetches the SHA of a tag from the remote repository.
func getRemoteTagSHA(tag string) (string, error) {
	if err := safeRef(tag); err != nil {
		return "", err
	}
	// Use git ls-remote to get the remote tag SHA (timeout-bounded, R4)
	output, err := gitNet("ls-remote", "--tags", "origin", "refs/tags/"+tag)
	if err != nil {
		slog.Error("failed to run git ls-remote", "error", err)
		return "", err
	}

	outputStr := strings.TrimSpace(string(output))
	if outputStr == "" {
		slog.Error("tag not found on remote", "tag", tag)
		return "", fmt.Errorf("tag %s not found on remote", tag)
	}

	// Parse the output (format: "SHA\trefs/tags/tagname")
	lines := strings.Split(outputStr, "\n")
	for _, line := range lines {
		parts := strings.Split(line, "\t")
		if len(parts) >= 2 && parts[1] == "refs/tags/"+tag {
			return parts[0], nil
		}
	}

	slog.Error("tag not found in remote output", "tag", tag)
	return "", fmt.Errorf("tag %s not found in remote output", tag)
}
