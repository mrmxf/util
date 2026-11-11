//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/

package bc

import (
	"os/exec"
	"strings"
	"testing"
)

// A failed git command must say why: the error carries the command and git's
// own message, not just "exit status N".
func TestGitNetRunErrorCarriesGitOutput(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	t.Chdir(t.TempDir())
	if out, err := exec.Command("git", "init", "-q").CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, out)
	}
	err := gitNetRun("checkout", "-q", "no-such-ref", "--")
	if err == nil {
		t.Fatal("checkout of a missing ref succeeded")
	}
	msg := err.Error()
	for _, want := range []string{"git checkout -q no-such-ref --", "exit status", "no-such-ref"} {
		if !strings.Contains(msg, want) {
			t.Errorf("error %q lacks %q", msg, want)
		}
	}
	if _, err := revParse("no-such-ref^{commit}"); err == nil {
		t.Error("revParse of a missing ref succeeded")
	}
}
