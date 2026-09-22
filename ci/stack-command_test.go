//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/

package ci

import (
	"bytes"
	"strings"
	"testing"
)

// runCI drives a sub-command the way a konfig snippet does, against a fixture
// config, and returns what it printed.
func runCI(t *testing.T, cfg Config, cmd string, args ...string) (string, error) {
	t.Helper()
	orig := LoadConfig
	LoadConfig = func() (Config, error) { return cfg, nil }
	t.Cleanup(func() { LoadConfig = orig })

	var out bytes.Buffer
	target := stackCmd
	if cmd == "scan" {
		target = scanCmd
	}
	target.SetOut(&out)
	target.SetErr(&out)
	if err := target.ParseFlags(args); err != nil {
		return out.String(), err
	}
	err := target.RunE(target, target.Flags().Args())
	return out.String(), err
}

func verbFixture() Config {
	return Config{
		Stack: StackList{
			{Name: "bonfire", Type: StackHugo},
			{Name: "form-parking", Type: StackGolang},
		},
		Targets: map[string]Target{
			"site":    {Kind: KindGitHubPages, Scan: ScanAxis{Vuln: VulnNone}, Prod: map[string]any{"dir": "kodata"}},
			"parking": {Kind: KindRegistry, Stack: "form-parking", Prod: map[string]any{"image": "acme/p:v1"}},
		},
	}
}

func resetStackFlags(t *testing.T) {
	t.Helper()
	stackSelectFlag, stackEchoModeFlag, stackEchoWatchFlag = "", "", false
	scanFormatFlag, scanTargetFlag, scanModeFlag, scanStackFlag = "env", "", "", ""
	scanListTargetsFlag = false
}

func TestStackCommandSurface(t *testing.T) {
	cfg := verbFixture()

	resetStackFlags(t)
	got, err := runCI(t, cfg, "stack")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Fields(got)[0] != "bonfire" {
		t.Errorf("`ci stack` = %q, want declaration order with bonfire first", got)
	}

	resetStackFlags(t)
	got, err = runCI(t, cfg, "stack", "get", "chk", "--stack", "bonfire")
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(got) != "pre-build lint scan" {
		t.Errorf("chk for bonfire = %q", got)
	}

	// The union across both stacks runs each phase once.
	resetStackFlags(t)
	got, err = runCI(t, cfg, "stack", "get", "chk")
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(got) != "pre-build lint scan test" {
		t.Errorf("chk union = %q", got)
	}
}

// The echo is what makes narrowing honest, so it is worth a test of its own.
func TestStackEchoCommand(t *testing.T) {
	cfg := verbFixture()

	resetStackFlags(t)
	got, err := runCI(t, cfg, "stack", "echo", "building", "--mode", "prod")
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(got) != "building bonfire, form-parking (prod)" {
		t.Errorf("unnarrowed echo = %q", got)
	}

	resetStackFlags(t)
	got, err = runCI(t, cfg, "stack", "echo", "building", "--stack", "bonfire", "--mode", "dev")
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(got) != "building bonfire (dev), ignoring [form-parking]" {
		t.Errorf("narrowed echo = %q", got)
	}

	// watch selection is the first stack, and says what it left out
	resetStackFlags(t)
	got, err = runCI(t, cfg, "stack", "echo", "watching", "--watch")
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(got) != "watching bonfire, ignoring [form-parking]" {
		t.Errorf("watch echo = %q", got)
	}
}

func TestScanCommandSurface(t *testing.T) {
	cfg := verbFixture()

	resetStackFlags(t)
	got, err := runCI(t, cfg, "scan", "--mode", "prod")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"scan_source_vuln=repo", "scan_source_secrets=true"} {
		if !strings.Contains(got, want) {
			t.Errorf("source env missing %q:\n%s", want, got)
		}
	}

	resetStackFlags(t)
	got, err = runCI(t, cfg, "scan", "--mode", "prod", "--target", "parking")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "scan_artifact_vuln=image") || !strings.Contains(got, "scan_artifact_ref=acme/p:v1") {
		t.Errorf("artifact env for the registry target:\n%s", got)
	}

	// The loop the scan check block runs.
	resetStackFlags(t)
	got, err = runCI(t, cfg, "scan", "--list-targets")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(strings.Fields(got), " ") != "parking site" {
		t.Errorf("--list-targets = %q", got)
	}
}
