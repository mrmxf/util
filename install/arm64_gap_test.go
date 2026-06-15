//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/

package install

import (
	"strings"
	"testing"
)

func TestAmd64Sibling(t *testing.T) {
	cases := map[Platform]Platform{
		PlatformLinuxDebArm64: PlatformLinuxDebAmd64,
		PlatformLinuxRpmArm64: PlatformLinuxRpmAmd64,
		PlatformLinuxDebAmd64: "",
		PlatformDarwinArm64:   "",
	}
	for in, want := range cases {
		if got := amd64Sibling(in); got != want {
			t.Errorf("amd64Sibling(%s)=%q want %q", in, got, want)
		}
	}
}

// In the test runner stdin is not a TTY, so resolveUnsupported must
// fail-unsupported regardless of a viable sibling (decision D15).
func TestResolveUnsupported_NonInteractiveFails(t *testing.T) {
	orig := stdinIsInteractive
	stdinIsInteractive = func() bool { return false }
	defer func() { stdinIsInteractive = orig }()

	entry := ToolEntry{
		Platforms: map[Platform]PlatformEntry{
			PlatformLinuxDebArm64: {Unsupported: true},
			PlatformLinuxDebAmd64: {Install: &InstallSpec{Strategy: "copy-binary", Dest: "/usr/bin/x"}},
		},
	}
	_, err := resolveUnsupported(entry, "x", PlatformLinuxDebArm64)
	if err == nil {
		t.Fatal("expected fail-unsupported in non-interactive shell")
	}
	if !strings.Contains(err.Error(), "not supported") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGoInstallImport(t *testing.T) {
	t.Run("top-level install", func(t *testing.T) {
		entry := ToolEntry{Install: &InstallSpec{Strategy: "go-install", Import: "example.com/x@latest"}}
		if got := goInstallImport(entry); got != "example.com/x@latest" {
			t.Errorf("got %q", got)
		}
	})
	t.Run("platform-level install", func(t *testing.T) {
		entry := ToolEntry{Platforms: map[Platform]PlatformEntry{
			PlatformLinuxDebAmd64: {Install: &InstallSpec{Strategy: "go-install", Import: "example.com/y@v1"}},
		}}
		if got := goInstallImport(entry); got != "example.com/y@v1" {
			t.Errorf("got %q", got)
		}
	})
	t.Run("none declared", func(t *testing.T) {
		entry := ToolEntry{Install: &InstallSpec{Strategy: "copy-binary"}}
		if got := goInstallImport(entry); got != "" {
			t.Errorf("expected empty, got %q", got)
		}
	})
}
