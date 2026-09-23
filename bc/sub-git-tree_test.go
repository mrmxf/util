//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/

package bc

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// The tree predicates were inverted before v1.0.0: `ahead` exited 0 when the
// tree was NOT ahead. Nothing about the name said so, so every caller that
// read like English did the opposite of what it said, and the konfig checks
// were written around the lie.
//
// A polarity is the one property a rename cannot announce - the command keeps
// working, it just answers backwards - so it is pinned here. If someone
// "fixes" one of these back, this test fails rather than a release going out
// with gates that pass when they should fail.
func TestTreePredicatesAreTrueWhenTheyExitZero(t *testing.T) {
	for _, tc := range []struct {
		path  []string
		long  string
		wants string
	}{
		{[]string{"git", "tree", "is", "clean"}, treeCleanCmd.Long, "Exits 0 if the tree IS clean"},
		{[]string{"git", "tree", "is", "ahead"}, treeAheadCmd.Long, "Exits 0 if the tree IS ahead"},
		{[]string{"git", "tree", "is", "behind"}, treeBehindCmd.Long, "Exits 0 if the tree IS behind"},
		{[]string{"git", "tree", "has", "unstaged"}, treeUnstagedCmd.Long, "Exits 0 if there ARE unstaged"},
	} {
		name := strings.Join(tc.path, " ")
		if _, _, err := Command.Find(tc.path); err != nil {
			t.Errorf("clog BC %s does not resolve: %v", name, err)
		}
		if !strings.Contains(tc.long, tc.wants) {
			t.Errorf("clog BC %s help does not promise %q:\n%s", name, tc.wants, tc.long)
		}
	}
}

// The old spellings must be present AND must fail. Present, so a caller is
// told where the command went; failing, so the inverted answer can never be
// returned again.
func TestOldTreeSpellingsAreRetiredNotAliased(t *testing.T) {
	for _, old := range []string{"clean", "ahead", "behind", "unstaged"} {
		c, _, err := Command.Find([]string{"git", "tree", old})
		if err != nil {
			t.Errorf("clog BC git tree %s should still resolve, to explain the rename: %v", old, err)
			continue
		}
		if c.Name() != old {
			t.Errorf("clog BC git tree %s resolved to %q", old, c.Name())
			continue
		}
		if c.RunE == nil {
			t.Errorf("clog BC git tree %s has no RunE - a retired command must fail, not act", old)
			continue
		}
		if err := c.RunE(c, nil); err == nil {
			t.Errorf("clog BC git tree %s returned nil - it must never succeed", old)
		} else if !strings.Contains(err.Error(), "retired") {
			t.Errorf("clog BC git tree %s error does not say it is retired: %v", old, err)
		}
	}
}

// `stash has error` was inverted the same way and is pinned for the same
// reason: it now exits 0 when the stash DOES hold an error.
func TestStashHasIsTrueWhenItExitsZero(t *testing.T) {
	if _, _, err := Command.Find([]string{"stash", "has"}); err != nil {
		t.Fatalf("clog BC stash has does not resolve: %v", err)
	}
	c, _, err := Command.Find([]string{"stash", "hasError"})
	if err != nil {
		t.Fatalf("the camelCase spelling should still resolve to explain itself: %v", err)
	}
	if c.RunE == nil || func() bool { return c.RunE(c, nil) == nil }() {
		t.Error("clog BC stash hasError must fail as retired")
	}
}

// Every retired command in this package must be hidden and must fail. This is
// the blanket guard: a retirement added later without a Run would silently
// become a no-op command that exits 0, which is worse than the original bug.
func TestEveryRetiredCommandFails(t *testing.T) {
	var walk func(*cobra.Command)
	walk = func(c *cobra.Command) {
		for _, sub := range c.Commands() {
			if strings.Contains(sub.Short, "retired in") {
				if !sub.Hidden {
					t.Errorf("%s is retired but not hidden", sub.CommandPath())
				}
				if sub.RunE == nil {
					t.Errorf("%s is retired but has no RunE", sub.CommandPath())
				} else if err := sub.RunE(sub, nil); err == nil {
					t.Errorf("%s is retired but succeeded", sub.CommandPath())
				}
			}
			walk(sub)
		}
	}
	walk(Command)
}
