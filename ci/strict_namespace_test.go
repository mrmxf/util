//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/

package ci

import (
	"io"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// A typo under a namespace used to print help and exit 0: cobra runs a
// grouping command's help Run with whatever words follow, so
// `clog CI <typo>` passed in every shell script that called it. Every
// command with subcommands must refuse a word that is not one of them.
func TestNamespaceTypoFails(t *testing.T) {
	Command.SetOut(io.Discard)
	Command.SetErr(io.Discard)
	t.Cleanup(func() { Command.SetArgs(nil) })

	var walk func(c *cobra.Command, path []string)
	walk = func(c *cobra.Command, path []string) {
		if c.HasSubCommands() {
			args := append(append([]string{}, path...), "zz-no-such-command")
			Command.SetArgs(args)
			if err := Command.Execute(); err == nil {
				t.Errorf("clog CI %s: exit 0, want an unknown-command error", strings.Join(args, " "))
			}
		}
		for _, s := range c.Commands() {
			// cobra's own completion and help, added when this package is the
			// root in a test; clog's real root supplies them in the binary.
			if s.Name() == "completion" || s.Name() == "help" {
				continue
			}
			walk(s, append(append([]string{}, path...), s.Name()))
		}
	}
	walk(Command, nil)
}
