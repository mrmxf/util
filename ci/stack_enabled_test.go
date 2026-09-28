//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/

package ci

import (
	"encoding/json"
	"strings"
	"testing"
)

// siteAndBox is a site plus a container stack whose registry target is
// prod-only; enabled is the container stack's `enabled:` as YAML would give it.
func siteAndBox(t *testing.T, enabled string) Config {
	t.Helper()
	var cfg Config
	doc := `{"stack":[{"name":"site","type":"hugo"},{"name":"box","type":"container"` +
		map[bool]string{true: `,"enabled":` + enabled, false: ``}[enabled != ""] + `}],
	  "targets":{
	    "pages":{"kind":"cloudflare-pages","stack":"site","prod":{"dir":"kodata"}},
	    "registry":{"kind":"container-registry","stack":"box","modes":["prod"],"prod":{"image":"a/b:{tag}"}}}}`
	if err := json.Unmarshal([]byte(doc), &cfg); err != nil {
		t.Fatalf("config: %v", err)
	}
	return cfg
}

func TestToggleValues(t *testing.T) {
	prev := getenv
	defer func() { getenv = prev }()
	env := map[string]string{"OFF": "false", "ON": "TRUE", "BAD": "maybe"}
	getenv = func(k string) string { return env[k] }

	for raw, want := range map[string]bool{
		`false`: false, `true`: true, `"no"`: false, `"0"`: false,
		`"$OFF"`: false, `"${ON}"`: true,
		`"$UNSET"`:          true, // an unset variable never switches anything off
		`"${UNSET:-false}"`: false, `"${OFF:-true}"`: false, `"${ON:-false}"`: true,
	} {
		var tg Toggle
		if err := json.Unmarshal([]byte(raw), &tg); err != nil {
			t.Fatalf("%s: %v", raw, err)
		}
		if got, _, err := tg.Value(); err != nil || got != want {
			t.Errorf("%s = %v, %v; want %v", raw, got, err, want)
		}
	}
	var bad Toggle
	_ = json.Unmarshal([]byte(`"$BAD"`), &bad)
	if _, _, err := bad.Value(); err == nil {
		t.Error(`"maybe" should be refused, not read as true or false`)
	}
	if on, _, _ := (Toggle{}).Value(); !on {
		t.Error("an unset toggle must be on")
	}
}

// Switched off, a stack drops out of every batch verb and takes its targets
// with it; named on the command line, it is refused rather than skipped.
func TestDisabledStackDropsOutEverywhere(t *testing.T) {
	cfg := siteAndBox(t, "false")

	selected, all, err := StackSelect(cfg, "")
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(StackNames(selected), ","); got != "site" {
		t.Errorf("selected = %s, want site only", got)
	}
	if got := strings.Join(StackMake(selected), " "); strings.Contains(got, "podman") {
		t.Errorf("make = %s, the disabled stack's podman must not run", got)
	}
	if got := StackEcho("building", selected, all, "dev"); got != "building site (dev), ignoring [box (enabled: false)]" {
		t.Errorf("echo = %q", got)
	}
	if _, _, err := StackSelect(cfg, "box"); err == nil || !strings.Contains(err.Error(), "switched off") {
		t.Errorf("naming a disabled stack = %v, want a refusal", err)
	}
	names, err := TargetNames(cfg, ModeProd, "")
	if err != nil || strings.Join(names, ",") != "pages" {
		t.Errorf("prod targets = %v, %v; want pages only", names, err)
	}
	scan, err := ScanTargets(cfg, selected, all)
	if err != nil || strings.Join(scan, ",") != "pages" {
		t.Errorf("scan targets = %v, %v; want pages only", scan, err)
	}
}

func TestDisabledByEnvironment(t *testing.T) {
	prev := getenv
	defer func() { getenv = prev }()
	getenv = func(k string) string { return map[string]string{"BUILD_CONTAINER_IMAGE": "false"}[k] }

	cfg := siteAndBox(t, `"$BUILD_CONTAINER_IMAGE"`)
	if names, _ := TargetNames(cfg, ModeProd, ""); strings.Join(names, ",") != "pages" {
		t.Errorf("BUILD_CONTAINER_IMAGE=false: targets = %v", names)
	}
	getenv = func(string) string { return "" }
	if names, _ := TargetNames(cfg, ModeProd, ""); strings.Join(names, ",") != "pages,registry" {
		t.Errorf("unset: targets = %v, want both", names)
	}
}

func TestBadToggleFailsAtParse(t *testing.T) {
	if _, err := Stacks(siteAndBox(t, `"sometimes"`)); err == nil || !strings.Contains(err.Error(), "enabled") {
		t.Errorf("err = %v, want it to name enabled", err)
	}
}
