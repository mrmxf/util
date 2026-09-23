//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/

package bc

import (
	"io"
	"strings"
	"testing"
)

// `clog BC semver <needs> <have>` printed help and exited 0, so every version
// check still written that way passed on any version at all.
func TestOldSemverFormFails(t *testing.T) {
	Command.SetOut(io.Discard)
	Command.SetErr(io.Discard)
	t.Cleanup(func() { Command.SetArgs(nil) })

	Command.SetArgs([]string{"semver", "9.9.9", "0.0.1"})
	if err := Command.Execute(); err == nil || !strings.Contains(err.Error(), "semver satisfies") {
		t.Fatalf("old form: want a retirement naming `semver satisfies`, got %v", err)
	}
	Command.SetArgs([]string{"semver", "satisfies", "0.1.0", "0.2.0"})
	if err := Command.Execute(); err != nil {
		t.Fatalf("`semver satisfies 0.1.0 0.2.0` should pass: %v", err)
	}
}
