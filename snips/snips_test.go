//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/

package snips

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func captureWarnings(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn})))
	t.Cleanup(func() { slog.SetDefault(prev) })
	return &buf
}

func TestSnippetOverridesBuiltin(t *testing.T) {
	root := &cobra.Command{Use: "clog"}
	root.AddCommand(&cobra.Command{Use: "Install"}, &cobra.Command{Use: "build", Run: func(*cobra.Command, []string) {}})

	recurseRawMap(root, SnippetGroup{}, 0, RawSnippets{
		"build":   "echo local build",
		"install": map[string]any{"htmltest": "echo htmltest"},
	})

	var builds, installs []*cobra.Command
	for _, c := range root.Commands() {
		switch c.Name() {
		case "build":
			builds = append(builds, c)
		case "Install", "install":
			installs = append(installs, c)
		}
	}

	if len(builds) != 1 || builds[0].Annotations["is-a"] != "snippet" {
		t.Fatalf("expected exactly one build command and it to be the snippet, got %d", len(builds))
	}
	if got := builds[0].Annotations[overridesKey]; got != "clog build" {
		t.Errorf("build overrides annotation = %q, want %q", got, "clog build")
	}
	// names are case-sensitive: the snippet group "install" must not replace "Install"
	if len(installs) != 2 {
		t.Errorf("expected Install and install to coexist, got %d commands", len(installs))
	}

	buf := captureWarnings(t)
	warnIfOverride(builds[0])
	if !strings.Contains(buf.String(), "overriding built-in command") {
		t.Errorf("expected override warning, got %q", buf.String())
	}
}

func TestNoWarningWithoutOverride(t *testing.T) {
	root := &cobra.Command{Use: "clog"}
	recurseRawMap(root, SnippetGroup{}, 0, RawSnippets{"grp": map[string]any{"leaf": "echo hi"}})

	leaf, _, err := root.Find([]string{"grp", "leaf"})
	if err != nil {
		t.Fatal(err)
	}
	buf := captureWarnings(t)
	warnIfOverride(leaf)
	if buf.Len() != 0 {
		t.Errorf("unexpected warning: %q", buf.String())
	}
}
