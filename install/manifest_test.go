//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/

package install

import (
	"io/fs"
	"path"
	"strings"
	"testing"
)

// index.yaml is the lookup, and a recipe that is not in it does not exist as
// far as `clog Install` is concerned.
//
// trivy shipped that way: recipes/trivy.yaml landed with the scanning work and
// was never indexed, so `clog Install trivy` failed with "not found in
// manifest" while the recipe sat right there in the binary. It broke the
// default tools list of five stack types at once, and it was found by a CI run
// rather than by a test.
//
// The check is bidirectional on purpose. One direction catches the recipe
// nobody indexed; the other catches the index entry whose file was renamed or
// deleted, which fails identically at the call site and is just as invisible
// from either file alone.
func TestEveryRecipeIsIndexedAndEveryIndexEntryExists(t *testing.T) {
	m, err := LoadManifest()
	if err != nil {
		t.Fatalf("cannot load index.yaml: %v", err)
	}

	onDisk := map[string]string{}
	err = fs.WalkDir(EmbedFS, "recipes", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(p, ".yaml") {
			return nil
		}
		onDisk[strings.TrimSuffix(path.Base(p), ".yaml")] = p
		return nil
	})
	if err != nil {
		t.Fatalf("cannot walk recipes/: %v", err)
	}
	if len(onDisk) == 0 {
		t.Fatal("no recipes found - the embed pattern is wrong and every tool would fail")
	}

	for name, file := range onDisk {
		if _, ok := m.Tools[name]; !ok {
			t.Errorf("%s exists but is not in index.yaml, so `clog Install %s` reports "+
				"\"not found in manifest\" - add it to index.yaml", file, name)
		}
	}

	for name, entry := range m.Tools {
		if entry.RecipePath == "" {
			t.Errorf("index.yaml lists %q with no recipepath", name)
			continue
		}
		if _, err := EmbedFS.Open(entry.RecipePath); err != nil {
			t.Errorf("index.yaml points %q at %s, which is not in the embedded fs: %v",
				name, entry.RecipePath, err)
		}
	}
}

// A recipe's top-level key is what the loader reads, so a file named one thing
// and keyed another resolves to nothing useful.
func TestRecipeFilenameMatchesItsKey(t *testing.T) {
	m, err := LoadManifest()
	if err != nil {
		t.Fatalf("cannot load index.yaml: %v", err)
	}
	for name, entry := range m.Tools {
		if entry.RecipePath == "" {
			continue
		}
		if base := strings.TrimSuffix(path.Base(entry.RecipePath), ".yaml"); base != name {
			t.Errorf("index.yaml key %q points at %s; the file should be named after the tool",
				name, entry.RecipePath)
		}
	}
}
