//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/

package ci

import (
	"strings"
	"testing"
)

func scanEnv() Env { return Env{Getenv: func(string) string { return "" }} }

// The source sweep needs no config: the honest default for a worktree is to
// look at all of it. This is the axis that covers a Go library, which has no
// deploy target at all and would otherwise never be scanned.
func TestSourceScanDefaults(t *testing.T) {
	a, err := SourceScan(Config{})
	if err != nil {
		t.Fatal(err)
	}
	if a.Vuln != VulnRepo || a.Ref != "." {
		t.Errorf("source sweep = %s %s, want the whole worktree", a.Vuln, a.Ref)
	}
	if !boolOr(a.Secrets, false) {
		t.Error("secret scanning should default on: it is the one sweep that works everywhere")
	}
	if !boolOr(a.IgnoreUnfixed, false) {
		t.Error("ignore-unfixed should default on, or a blocking gate goes chronically red")
	}
	if a.Severity != "HIGH,CRITICAL" {
		t.Errorf("severity = %q", a.Severity)
	}

	bad := Config{Scan: ScanConfig{Source: ScanAxis{Vuln: "everything"}}}
	if _, err := SourceScan(bad); err == nil || !strings.Contains(err.Error(), "not one of") {
		t.Errorf("an unknown sweep should be rejected, got %v", err)
	}
}

func TestArtifactScanKindDefaults(t *testing.T) {
	cfg := Config{
		Stack: StackList{{Name: "app", Type: StackContainer}},
		Targets: map[string]Target{
			"registry": {Kind: KindRegistry, Prod: map[string]any{"image": "acme/site:v1"}},
			"release":  {Kind: KindGitHubRelease, Prod: map[string]any{"dir": "tmp/firmware"}},
		},
	}
	for _, tc := range []struct{ target, vuln, ref string }{
		{"registry", VulnImage, "acme/site:v1"},
		{"release", VulnFS, "tmp/firmware"},
	} {
		a, err := ArtifactScan(scanEnv(), cfg, ModeProd, tc.target)
		if err != nil {
			t.Fatalf("%s: %v", tc.target, err)
		}
		if a.Vuln != tc.vuln || a.Ref != tc.ref {
			t.Errorf("%s = %s %s, want %s %s", tc.target, a.Vuln, a.Ref, tc.vuln, tc.ref)
		}
		if a.Severity != "CRITICAL" {
			t.Errorf("%s severity = %q, want the artifact default", tc.target, a.Severity)
		}
	}
}

// A rendered site has no lockfiles, so a filesystem sweep of it finds nothing
// and reports that in green. There is no honest default, and silence must not
// mean "not scanned" - so the target has to write it down.
func TestPagesKindsRequireAnExplicitSweep(t *testing.T) {
	cfg := Config{Targets: map[string]Target{
		"pages": {Kind: KindGitHubPages, Prod: map[string]any{"dir": "kodata"}},
	}}
	_, err := ArtifactScan(scanEnv(), cfg, ModeProd, "pages")
	if err == nil {
		t.Fatal("a Pages target with no sweep should be an error")
	}
	for _, want := range []string{"pages", "vuln: none", "github-pages"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the error should mention %q, got: %v", want, err)
		}
	}

	// Written down, it is accepted and nothing is resolved for it.
	cfg.Targets["pages"] = Target{Kind: KindGitHubPages, Scan: ScanAxis{Vuln: VulnNone},
		Prod: map[string]any{"dir": "kodata"}}
	a, err := ArtifactScan(scanEnv(), cfg, ModeProd, "pages")
	if err != nil {
		t.Fatal(err)
	}
	if a.Vuln != VulnNone {
		t.Errorf("vuln = %q, want none", a.Vuln)
	}
	if !boolOr(a.Secrets, false) {
		t.Error("secrets should still default on: a key baked into a built asset is the real risk here")
	}
}

// Most specific wins: the target's own block beats ci.scan.artifact.
func TestArtifactScanOverrideLadder(t *testing.T) {
	no := false
	cfg := Config{
		Scan: ScanConfig{Artifact: ScanAxis{Severity: "HIGH,CRITICAL", IgnoreUnfixed: &no}},
		Targets: map[string]Target{
			"a": {Kind: KindRegistry, Prod: map[string]any{"image": "acme/a"}},
			"b": {Kind: KindRegistry, Scan: ScanAxis{Severity: "CRITICAL"}, Prod: map[string]any{"image": "acme/b"}},
		},
	}
	a, err := ArtifactScan(scanEnv(), cfg, ModeProd, "a")
	if err != nil {
		t.Fatal(err)
	}
	if a.Severity != "HIGH,CRITICAL" || boolOr(a.IgnoreUnfixed, true) {
		t.Errorf("repo-wide artifact defaults should apply, got %q unfixed=%v", a.Severity, boolOr(a.IgnoreUnfixed, true))
	}
	b, err := ArtifactScan(scanEnv(), cfg, ModeProd, "b")
	if err != nil {
		t.Fatal(err)
	}
	if b.Severity != "CRITICAL" {
		t.Errorf("the target's own severity should win, got %q", b.Severity)
	}
}

// dir and image live inside the dev:/prod: maps, not on Target, so resolution
// is mode-aware and a dev build asking for a prod-only dir fails by name.
func TestArtifactScanRefIsModeAware(t *testing.T) {
	cfg := Config{Targets: map[string]Target{
		"pages": {Kind: KindGitHubRelease, Prod: map[string]any{"dir": "kodata"}},
	}}
	_, err := ArtifactScan(scanEnv(), cfg, ModeDev, "pages")
	if err == nil || !strings.Contains(err.Error(), "dev") {
		t.Errorf("a prod-only target in dev mode should fail naming the mode, got %v", err)
	}
}

// The headline regression guard. `clog CI target list` filters by deploy-mode
// membership, so a prod-only target reports no dev membership - enumerating
// scans with it would leave exactly those targets unscanned on every pull
// request, which is the original bug through a different door.
func TestScanTargetsIgnoresDeployModeMembership(t *testing.T) {
	cfg := Config{
		Stack: StackList{{Name: "site", Type: StackHugo}},
		Targets: map[string]Target{
			"prod-only": {Kind: KindRegistry, Modes: []string{ModeProd},
				Prod: map[string]any{"image": "acme/site"}},
		},
	}
	// TargetNames, which deploy uses, correctly sees nothing in dev.
	deploying, err := TargetNames(cfg, ModeDev, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(deploying) != 0 {
		t.Fatalf("precondition: a prod-only target should not deploy in dev, got %v", deploying)
	}
	// The scan must still see it.
	all, err := Stacks(cfg)
	if err != nil {
		t.Fatal(err)
	}
	scanned, err := ScanTargets(cfg, all, all)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(scanned, " ") != "prod-only" {
		t.Errorf("scan targets = %v, want the prod-only target scanned on a dev build too", scanned)
	}
}

func TestScanEnvLines(t *testing.T) {
	a, err := SourceScan(Config{})
	if err != nil {
		t.Fatal(err)
	}
	out := a.EnvLines("scan_source")
	for _, want := range []string{"scan_source_vuln=repo", "scan_source_secrets=true", "scan_source_ignore_unfixed=true"} {
		if !strings.Contains(out, want) {
			t.Errorf("env lines missing %q:\n%s", want, out)
		}
	}
}
