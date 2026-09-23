//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/

package ci

import (
	"strings"
	"testing"
)

// Bare `stack` and `scan` printed values before v1.0.0. As plain namespaces
// they printed help and exited 0, so an old caller got help text as data.
func TestRetiredNamespacesFailLoudly(t *testing.T) {
	for cmd, use := range map[string]string{
		"stack": "clog CI stack list",
		"scan":  "clog CI scan show",
	} {
		_, err := runCI(t, Config{}, cmd)
		if err == nil || !strings.Contains(err.Error(), "use: "+use) {
			t.Errorf("clog CI %s: want a retirement naming %q, got %v", cmd, use, err)
		}
	}
}
