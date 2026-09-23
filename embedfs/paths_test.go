//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/

package embedfs

import (
	"io/fs"
	"regexp"
	"strings"
	"testing"
)

// The canonical artifact layout exists so a debugger looks in a named place
// rather than knowing that binaries were in tmp/, the check log in
// tmp/BC-stash.yaml, SARIF in tmp/scan/ and a rendered site in kodata/.
//
// That only holds while nothing writes anywhere else, and the drift back is
// easy: `mkdir -p tmp` is three words and looks harmless in a diff.
var writesOutsideCanonical = regexp.MustCompile(
	`(mkdir -p|--output|--tarball=|-o |>\s*)\s*"?\.?/?tmp/`)

func TestKonfigWritesOnlyToCanonicalDirs(t *testing.T) {
	err := fs.WalkDir(CoreFs, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".yaml") {
			return err
		}
		b, rerr := fs.ReadFile(CoreFs, p)
		if rerr != nil {
			return rerr
		}
		for i, line := range strings.Split(string(b), "\n") {
			if strings.Contains(line, "legacy") || strings.HasPrefix(strings.TrimSpace(line), "#") {
				continue
			}
			if writesOutsideCanonical.MatchString(line) {
				t.Errorf("%s:%d writes outside _clog_build/ or _clog_deploy/:\n  %s",
					p, i+1, strings.TrimSpace(line))
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("cannot walk the embedded konfig: %v", err)
	}
}
