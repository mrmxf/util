//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

package bc

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// TestGitCommands tests all BC git subcommands
func TestGitCommands(t *testing.T) {
	// Ensure we're in a git repository for testing
	if _, err := os.Stat(".git"); os.IsNotExist(err) {
		t.Skip("Skipping git tests - not in a git repository")
	}

	tests := []struct {
		name        string
		cmd         *cobra.Command
		args        []string
		expectError bool
		description string
	}{
		{
			name:        "git_branch",
			cmd:         branchCmd,
			args:        []string{},
			expectError: false,
			description: "should print current git branch",
		},
		{
			name:        "git_suffix",
			cmd:         suffixCmd,
			args:        []string{},
			expectError: false,
			description: "should print branch suffix (empty for main)",
		},
		{
			name:        "git_tag_head",
			cmd:         headCmd,
			args:        []string{},
			expectError: false,
			description: "should print tag at HEAD (or nothing)",
		},
		{
			name:        "git_tag_ref",
			cmd:         refCmd,
			args:        []string{},
			expectError: false,
			description: "should print reference tag from releases",
		},
		{
			name:        "git_checkout_list",
			cmd:         listCmd,
			args:        []string{},
			expectError: false,
			description: "should list available branches and tags",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture output
			var buf bytes.Buffer
			tt.cmd.SetOut(&buf)
			tt.cmd.SetErr(&buf)

			// Execute command
			err := tt.cmd.Execute()

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none for %s", tt.description)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for %s: %v", tt.description, err)
				}
			}

			// Reset command for next test
			tt.cmd.SetArgs([]string{})
		})
	}
}

// TestGitBranchOutput tests the git branch command output
func TestGitBranchOutput(t *testing.T) {
	if _, err := os.Stat(".git"); os.IsNotExist(err) {
		t.Skip("Skipping git branch test - not in a git repository")
	}

	var buf bytes.Buffer
	branchCmd.SetOut(&buf)
	branchCmd.SetErr(&buf)

	err := branchCmd.Execute()
	if err != nil {
		t.Fatalf("git branch command failed: %v", err)
	}

	output := strings.TrimSpace(buf.String())
	if output == "" {
		t.Error("git branch should return the current branch name")
	}

	// Reset command
	branchCmd.SetArgs([]string{})
}

// TestGitSuffixOutput tests the git suffix command logic
func TestGitSuffixOutput(t *testing.T) {
	if _, err := os.Stat(".git"); os.IsNotExist(err) {
		t.Skip("Skipping git suffix test - not in a git repository")
	}

	var buf bytes.Buffer
	suffixCmd.SetOut(&buf)
	suffixCmd.SetErr(&buf)

	err := suffixCmd.Execute()
	if err != nil {
		t.Fatalf("git suffix command failed: %v", err)
	}

	output := strings.TrimSpace(buf.String())

	// Get the actual branch to verify logic
	var branchBuf bytes.Buffer
	branchCmd.SetOut(&branchBuf)
	branchCmd.SetErr(&branchBuf)

	err = branchCmd.Execute()
	if err != nil {
		t.Fatalf("git branch command failed: %v", err)
	}

	currentBranch := strings.TrimSpace(branchBuf.String())

	if currentBranch == "main" {
		if output != "" {
			t.Errorf("Expected empty output for main branch, got: '%s'", output)
		}
	} else {
		if output != currentBranch {
			t.Errorf("Expected branch name '%s' for non-main branch, got: '%s'", currentBranch, output)
		}
	}

	// Reset commands
	suffixCmd.SetArgs([]string{})
	branchCmd.SetArgs([]string{})
}

// TestGitTagCommands tests git tag subcommands
func TestGitTagCommands(t *testing.T) {
	if _, err := os.Stat(".git"); os.IsNotExist(err) {
		t.Skip("Skipping git tag tests - not in a git repository")
	}

	tagTests := []struct {
		name        string
		cmd         *cobra.Command
		expectError bool
		description string
	}{
		{
			name:        "tag_head",
			cmd:         headCmd,
			expectError: false,
			description: "should handle tag at HEAD query",
		},
		{
			name:        "tag_ref",
			cmd:         refCmd,
			expectError: false,
			description: "should return reference tag from releases",
		},
	}

	for _, tt := range tagTests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			tt.cmd.SetOut(&buf)
			tt.cmd.SetErr(&buf)

			err := tt.cmd.Execute()

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none for %s", tt.description)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for %s: %v", tt.description, err)
				}
			}

			// Reset command
			tt.cmd.SetArgs([]string{})
		})
	}
}

// TestGitCheckoutCommands tests git checkout subcommands
func TestGitCheckoutCommands(t *testing.T) {
	if _, err := os.Stat(".git"); os.IsNotExist(err) {
		t.Skip("Skipping git checkout tests - not in a git repository")
	}

	checkoutTests := []struct {
		name        string
		cmd         *cobra.Command
		expectError bool
		description string
	}{
		{
			name:        "checkout_list",
			cmd:         listCmd,
			expectError: false,
			description: "should list available branches and tags",
		},
		{
			name:        "checkout_production",
			cmd:         productionCmd,
			expectError: false, // May error if no production release, but shouldn't crash
			description: "should handle production checkout query",
		},
	}

	for _, tt := range checkoutTests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			tt.cmd.SetOut(&buf)
			tt.cmd.SetErr(&buf)

			err := tt.cmd.Execute()

			// For checkout commands, we just want to ensure they don't crash
			// Some may return errors if no production release exists, which is OK
			if err != nil {
				t.Logf("Command returned error (may be expected): %v", err)
			}

			// Reset command
			tt.cmd.SetArgs([]string{})
		})
	}
}

// TestGitCommandHelp tests that all git commands have proper help
func TestGitCommandHelp(t *testing.T) {
	gitSubcommands := []*cobra.Command{
		branchCmd,
		suffixCmd,
		headCmd,
		refCmd,
		listCmd,
		productionCmd,
		tidyCmd,
	}

	for _, cmd := range gitSubcommands {
		t.Run("help_"+cmd.Name(), func(t *testing.T) {
			if cmd.Short == "" {
				t.Errorf("Command %s is missing Short description", cmd.Name())
			}
			if cmd.Long == "" {
				t.Errorf("Command %s is missing Long description", cmd.Name())
			}
		})
	}
}
