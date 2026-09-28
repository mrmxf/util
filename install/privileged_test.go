//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/

package install

import (
	"strings"
	"testing"
)

// A container CI job is root with no sudo binary; a laptop is not root. A
// recipe's sudo: true must work in both, so it means "needs root", not "call sudo".
func TestPrivilegedArgv(t *testing.T) {
	for _, c := range []struct {
		sudo bool
		euid int
		want string
	}{
		{true, 0, "apt-get install -y podman"},         // root in a container: no sudo
		{true, 1000, "sudo apt-get install -y podman"}, // a laptop or a CI VM
		{false, 1000, "apt-get install -y podman"},     // recipe needs no root
	} {
		cmd, argv := privilegedArgv(c.sudo, c.euid, "apt-get", "install", "-y", "podman")
		if got := strings.Join(append([]string{cmd}, argv...), " "); got != c.want {
			t.Errorf("sudo=%v euid=%d: %q, want %q", c.sudo, c.euid, got, c.want)
		}
	}
}
