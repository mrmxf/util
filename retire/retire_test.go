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
