//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/

package kfg

import (
	"os"
	"testing"
)

func TestMergedPathsAndFileHasKey(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := os.WriteFile(".clog.yaml", []byte("snippets:\n  install:\n    x: echo x\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := Konfigure(&KonfigureOpt{PreventAutoMerge: true}); err != nil {
		t.Fatal(err)
	}
	if err := MergeKonfig(&KonfigureOpt{FilePath: ".clog.yaml"}); err != nil {
		t.Fatal(err)
	}
	if err := MergeKonfig(&KonfigureOpt{FilePath: "missing.yaml"}); err != nil {
		t.Fatal(err)
	}

	if len(MergedPaths) != 1 || MergedPaths[0] != ".clog.yaml" {
		t.Errorf("MergedPaths = %v, want [.clog.yaml]", MergedPaths)
	}
	if !FileHasKey(".clog.yaml", "snippets.install") {
		t.Error("FileHasKey(snippets.install) = false, want true")
	}
	if FileHasKey(".clog.yaml", "snippets.getapp") || FileHasKey("missing.yaml", "snippets") {
		t.Error("FileHasKey reported a key that is not there")
	}

	if err := Konfigure(&KonfigureOpt{PreventAutoMerge: true}); err != nil {
		t.Fatal(err)
	}
	if len(MergedPaths) != 0 {
		t.Errorf("Konfigure should reset MergedPaths, got %v", MergedPaths)
	}
}
