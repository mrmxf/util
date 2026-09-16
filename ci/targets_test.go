//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/

package ci

import (
	"strings"
	"testing"
)

// twoTargetConfig is a website with two destinations: a registry in both modes
// and Cloudflare Pages in prod only.
func twoTargetConfig() Config {
	return Config{Targets: map[string]Target{
		"registry": {Kind: KindRegistry, Require: []string{"DOCKER_PAT"},
			Dev:  map[string]any{"image": "acme/site", "tags": []any{"dev"}},
			Prod: map[string]any{"image": "acme/site", "tags": []any{"{tag}", "latest"}}},
		"pages": {Kind: KindCloudflarePage, Modes: []string{ModeProd},
			Prod: map[string]any{"project": "site", "domain": "example.com"}},
	}}
}

func TestTargetNames(t *testing.T) {
	cfg := twoTargetConfig()
	for _, tc := range []struct {
		mode, kind string
		want       string
	}{
		{ModeDev, "", "registry"},
		{ModeProd, "", "pages registry"},
		{ModeProd, KindRegistry, "registry"},
		{ModeDev, KindCloudflarePage, ""},
	} {
		got, err := TargetNames(cfg, tc.mode, tc.kind)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Join(got, " ") != tc.want {
			t.Errorf("TargetNames(%s, %q) = %v, want %q", tc.mode, tc.kind, got, tc.want)
		}
	}

	bad := Config{Targets: map[string]Target{"x": {Kind: "ftp", Dev: map[string]any{}}}}
	if _, err := TargetNames(bad, ModeDev, ""); err == nil || !strings.Contains(err.Error(), "not one of") {
		t.Errorf("an unknown kind should be rejected, got %v", err)
	}
	none := Config{Targets: map[string]Target{"x": {Dev: map[string]any{}}}}
	if _, err := TargetNames(none, ModeDev, ""); err == nil || !strings.Contains(err.Error(), "kind is not set") {
		t.Errorf("a missing kind should be rejected, got %v", err)
	}
}

func TestTargetGet(t *testing.T) {
	saved := ReleaseVersion
	ReleaseVersion = func() string { return "v0.11.4" }
	t.Cleanup(func() { ReleaseVersion = saved })

	cfg := twoTargetConfig()
	env := fakeEnv(map[string]string{"GITHUB_SHA": "cafef00d"}, "")

	if got, _ := TargetGet(env, cfg, ModeProd, "registry", "tags", false); got != "v0.11.4\nlatest" {
		t.Errorf("prod tags = %q; lists print one per line and {tag} expands", got)
	}
	if got, _ := TargetGet(env, cfg, ModeDev, "registry", "image", false); got != "acme/site" {
		t.Errorf("dev image = %q", got)
	}
	if got, _ := TargetGet(env, cfg, ModeProd, "registry", "kind", false); got != KindRegistry {
		t.Errorf("kind = %q", got)
	}
	if got, _ := TargetGet(env, cfg, ModeProd, "registry", "nothing", false); got != "" {
		t.Errorf("a missing key prints nothing, got %q", got)
	}
	if _, err := TargetGet(env, cfg, ModeProd, "registry", "nothing", true); err == nil {
		t.Error("--required should fail on a missing key")
	}
	if _, err := TargetGet(env, cfg, ModeDev, "pages", "project", false); err == nil || !strings.Contains(err.Error(), "cannot deploy in dev") {
		t.Errorf("pages has no dev block: %v", err)
	}
	if _, err := TargetGet(env, cfg, ModeDev, "", "image", false); err == nil || !strings.Contains(err.Error(), TargetVar) {
		t.Errorf("an unset target should name $%s: %v", TargetVar, err)
	}
	if _, err := TargetGet(env, cfg, ModeDev, "nope", "image", false); err == nil || !strings.Contains(err.Error(), "no ci.targets.nope") {
		t.Errorf("an unknown target should say so: %v", err)
	}
}

func TestExpandTokens(t *testing.T) {
	saved := ReleaseVersion
	ReleaseVersion = func() string { return "v1.2.3" }
	t.Cleanup(func() { ReleaseVersion = saved })

	env := fakeEnv(map[string]string{"CI_COMMIT_SHA": "deadbeef"}, "")
	got := expandTokens("bin/{tag}/{version}/{mode}/{sha}", env, ModeProd)
	if got != "bin/v1.2.3/1.2.3/prod/deadbeef" {
		t.Errorf("expandTokens = %q", got)
	}
	if got := expandTokens("no tokens here", env, ModeDev); got != "no tokens here" {
		t.Errorf("unchanged text got mangled: %q", got)
	}
}

func TestModeGet(t *testing.T) {
	saved := ModeData
	ModeData = func(mode string) map[string]any {
		return map[string]map[string]any{
			ModeDev:  {"base-url": "http://localhost:1313/", "hugo-flags": "--buildDrafts"},
			ModeProd: {"base-url": "https://example.com/"},
		}[mode]
	}
	t.Cleanup(func() { ModeData = saved })

	if got, _ := ModeGet(ModeDev, "hugo-flags", false); got != "--buildDrafts" {
		t.Errorf("dev hugo-flags = %q", got)
	}
	if got, _ := ModeGet(ModeProd, "hugo-flags", false); got != "" {
		t.Errorf("prod has no hugo-flags, got %q", got)
	}
	if _, err := ModeGet(ModeProd, "hugo-flags", true); err == nil {
		t.Error("--required should fail on a missing mode setting")
	}
}
