//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/

package install

import "testing"

// TestRecipesCoverAllPlatforms enforces Correction 4 of the Phase 9 plan:
// every tool must explicitly cover all six supported platforms. A platform is
// "covered" when it is either marked unsupported, or has an install strategy
// (at the platform level or inherited from the entry level). A missing or empty
// platform entry is a failure, not a silent skip.
func TestRecipesCoverAllPlatforms(t *testing.T) {
	m, err := LoadManifest()
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}
	if len(m.Tools) == 0 {
		t.Fatal("manifest has no tools")
	}

	for name := range m.Tools {
		_, entry, err := LoadTool(m, name, nil)
		if err != nil {
			t.Errorf("%s: LoadTool: %v", name, err)
			continue
		}
		for _, p := range AllPlatforms {
			pe, ok := entry.Platforms[p]
			if !ok {
				t.Errorf("%s: missing platform %s (all six must be explicit)", name, p)
				continue
			}
			if pe.Unsupported {
				continue
			}
			if pe.Install == nil && entry.Install == nil {
				t.Errorf("%s/%s: no install strategy and not marked unsupported", name, p)
			}
		}
	}
}

// TestRecipeChecksParse ensures every tool's check: block parses into the
// Phase 8 check.Block schema and has at least one condition.
func TestRecipeChecksParse(t *testing.T) {
	m, err := LoadManifest()
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}
	for name := range m.Tools {
		_, entry, err := LoadTool(m, name, nil)
		if err != nil {
			t.Errorf("%s: LoadTool: %v", name, err)
			continue
		}
		if entry.Check == nil {
			t.Errorf("%s: no check block", name)
			continue
		}
		if len(entry.Check.Try) == 0 && len(entry.Check.Env) == 0 {
			t.Errorf("%s: check block has no env/try conditions", name)
		}
	}
}
