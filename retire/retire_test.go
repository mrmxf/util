//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/

package retire

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// A retired command must FAIL. The whole point is that it never quietly does
// the old thing, so this is the test that matters most.
func TestRetiredCommandAlwaysFails(t *testing.T) {
	root := &cobra.Command{Use: "clog"}
	ci := &cobra.Command{Use: "CI"}
	root.AddCommand(ci)
	Add(ci, "policy", "clog CI show policy", BareNoun)

	root.SetArgs([]string{"CI", "policy"})
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})

	err := root.Execute()
	if err == nil {
		t.Fatal("retired command returned nil error - it must never succeed")
	}
	for _, want := range []string{"clog CI policy", Version, BareNoun, "clog CI show policy"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error is missing %q:\n%s", want, err)
		}
	}
}

// Arguments must not change the answer. A caller typing the old command with
// its old flags should be told about the rename, not about arity.
func TestRetiredCommandIgnoresArgs(t *testing.T) {
	root := &cobra.Command{Use: "clog"}
	Add(root, "targets", "clog CI target list", BareNoun)

	root.SetArgs([]string{"targets", "--kind", "bucket", "extra"})
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})

	err := root.Execute()
	if err == nil {
		t.Fatal("retired command with args returned nil error")
	}
	if !strings.Contains(err.Error(), "clog CI target list") {
		t.Errorf("error should name the replacement, got:\n%s", err)
	}
}

// Retired commands are hidden: help output should describe the grammar that is
// current, not the one that was removed.
func TestRetiredCommandIsHidden(t *testing.T) {
	c := Command("policy", "clog CI show policy", BareNoun)
	if !c.Hidden {
		t.Error("retired command should be hidden from help")
	}
}

// The case Namespace exists for: cobra runs a parent with no RunE as help and
// exits 0, so an old positional form silently "passed".
func TestNamespaceOldFormFails(t *testing.T) {
	for _, args := range [][]string{
		{"semver"},
		{"semver", "0.1.0", "0.2.0"},
		{"semver", "--format", "env"},
	} {
		root, ran := namespaceTree()
		root.SetArgs(args)
		err := root.Execute()
		if err == nil || !strings.Contains(err.Error(), "use: clog semver satisfies") {
			t.Errorf("%v: want the retirement error, got %v", args, err)
		}
		if *ran {
			t.Errorf("%v: the subcommand ran", args)
		}
	}
}

func TestNamespaceSubcommandAndHelpStillWork(t *testing.T) {
	root, ran := namespaceTree()
	root.SetArgs([]string{"semver", "satisfies", "0.1.0", "0.2.0"})
	if err := root.Execute(); err != nil || !*ran {
		t.Fatalf("subcommand: err=%v ran=%v", err, *ran)
	}

	root, _ = namespaceTree()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"semver", "--help"})
	if err := root.Execute(); err != nil || !strings.Contains(out.String(), "satisfies") {
		t.Fatalf("--help: err=%v out=%q", err, out.String())
	}
}

func namespaceTree() (*cobra.Command, *bool) {
	ran := false
	root := &cobra.Command{Use: "clog", SilenceErrors: true}
	ns := &cobra.Command{Use: "semver", Run: func(*cobra.Command, []string) {}}
	ns.AddCommand(&cobra.Command{
		Use:  "satisfies",
		Args: cobra.ExactArgs(2),
		Run:  func(*cobra.Command, []string) { ran = true },
	})
	root.AddCommand(ns)
	Namespace(ns, "clog semver satisfies <needs> <have>", "why")
	var sink bytes.Buffer
	root.SetErr(&sink)
	return root, &ran
}
