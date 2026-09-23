//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/

package snips

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// tree builds a root with the named children (and grandchildren under "CI").
func tree(names ...string) *cobra.Command {
	root := &cobra.Command{Use: "clog"}
	for _, n := range names {
		root.AddCommand(&cobra.Command{Use: n, Run: func(*cobra.Command, []string) {}})
	}
	return root
}

// The load-bearing case: with no override, either case reaches the shipped
// command, so nobody has to hold shift.
func TestFoldsToShippedWhenUnambiguous(t *testing.T) {
	root := tree("Install", "Check", "CI")
	for _, tc := range []struct{ in, want string }{
		{"install", "Install"},
		{"INSTALL", "Install"},
		{"Install", "Install"},
		{"check", "Check"},
		{"ci", "CI"},
	} {
		got := ResolveCase(root, []string{tc.in})[0]
		if got != tc.want {
			t.Errorf("ResolveCase(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// The A/B pair. This is the one that must never regress: when both halves
// exist, each case reaches its own command and neither is rewritten.
func TestExactMatchWinsSoThePairSurvives(t *testing.T) {
	root := tree("Build", "build")
	if got := ResolveCase(root, []string{"build"})[0]; got != "build" {
		t.Errorf("lowercase `build` resolved to %q - the override is unreachable", got)
	}
	if got := ResolveCase(root, []string{"Build"})[0]; got != "Build" {
		t.Errorf("capitalised `Build` resolved to %q - the shipped default is unreachable", got)
	}
}

// An ambiguous fold is not what anyone typed, so the promise that the
// Capitalised name is always reachable decides it.
func TestAmbiguousFoldPrefersShipped(t *testing.T) {
	root := tree("Build", "build")
	if got := ResolveCase(root, []string{"BUILD"})[0]; got != "Build" {
		t.Errorf("ambiguous fold resolved to %q, want the shipped `Build`", got)
	}
}

// Resolution descends, so a sub-command in the wrong case also lands.
func TestResolvesNestedPath(t *testing.T) {
	root := &cobra.Command{Use: "clog"}
	ci := &cobra.Command{Use: "CI"}
	show := &cobra.Command{Use: "show", Run: func(*cobra.Command, []string) {}}
	ci.AddCommand(show)
	root.AddCommand(ci)

	got := ResolveCase(root, []string{"ci", "SHOW", "policy"})
	if got[0] != "CI" || got[1] != "show" {
		t.Errorf("ResolveCase = %v, want [CI show policy]", got)
	}
	// `policy` names no command, so it is left for `show` to interpret
	if got[2] != "policy" {
		t.Errorf("trailing arg was rewritten: %v", got)
	}
}

// Flags end the command path; nothing after one may be rewritten.
func TestStopsAtFirstFlag(t *testing.T) {
	root := tree("Install")
	got := ResolveCase(root, []string{"install", "--recipe", "Install"})
	if got[0] != "Install" {
		t.Errorf("command not resolved: %v", got)
	}
	if got[2] != "Install" {
		t.Errorf("flag value must not be rewritten: %v", got)
	}
}

// An unknown name is left alone so cobra can produce its own error.
func TestUnknownIsUntouched(t *testing.T) {
	root := tree("Install")
	if got := ResolveCase(root, []string{"nope"})[0]; got != "nope" {
		t.Errorf("unknown arg rewritten to %q", got)
	}
}

// Aliases participate, in both cases.
func TestAliasesFold(t *testing.T) {
	root := &cobra.Command{Use: "clog"}
	root.AddCommand(&cobra.Command{
		Use: "Buildinfo", Aliases: []string{"Version"},
		Run: func(*cobra.Command, []string) {},
	})
	if got := ResolveCase(root, []string{"version"})[0]; got != "Buildinfo" {
		t.Errorf("alias fold = %q, want Buildinfo", got)
	}
}

// Prevention (f): the case convention must not rot. Every shipped command in a
// clog tree is Capitalised, so a lowercase sibling is always someone's
// override and never an accident of naming.
func TestShippedNamesAreCapitalised(t *testing.T) {
	shipped := []string{"BC", "CI", "Check", "Install", "Log", "Crayon", "Source", "Snippets", "Buildinfo"}
	for _, n := range shipped {
		if !isShipped(&cobra.Command{Use: n}) {
			t.Errorf("shipped command %q is not Capitalised - the case rule says compiled-in names are", n)
		}
	}
	for _, n := range []string{"build", "watch", "deploy", "bc-golang"} {
		if isShipped(&cobra.Command{Use: n}) {
			t.Errorf("%q reads as shipped but is lowercase", n)
		}
	}
	// a name with a leading digit or symbol is neither, and must not panic
	if isShipped(&cobra.Command{Use: ""}) {
		t.Error("empty name must not report as shipped")
	}
}

// The resolver must never lose or reorder arguments.
func TestArgCountPreserved(t *testing.T) {
	root := tree("Install")
	in := []string{"install", "trivy", "--verbose"}
	got := ResolveCase(root, in)
	if len(got) != len(in) {
		t.Fatalf("arg count changed: %v -> %v", in, got)
	}
	if strings.Join(got[1:], " ") != "trivy --verbose" {
		t.Errorf("args after the command changed: %v", got)
	}
}
