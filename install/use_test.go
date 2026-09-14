//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/

package install

import "testing"

func TestNormaliseVersion(t *testing.T) {
	cases := []struct{ req, prefix, want string }{
		{"1.26.4", "go", "go1.26.4"},
		{"go1.26.4", "go", "go1.26.4"},
		{"v1.26.4", "go", "go1.26.4"},
		{"0.19.1", "", "0.19.1"},
		{"v0.19.1", "", "0.19.1"},
	}
	for _, c := range cases {
		got, err := normaliseVersion(c.req, c.prefix)
		if err != nil || got != c.want {
			t.Errorf("normaliseVersion(%q,%q) = %q,%v want %q", c.req, c.prefix, got, err, c.want)
		}
	}
	for _, bad := range []string{"", "v", "stable", "go"} {
		if _, err := normaliseVersion(bad, "go"); err == nil {
			t.Errorf("normaliseVersion(%q) expected error", bad)
		}
	}
}

func TestPickLTS(t *testing.T) {
	var tags []versionTag
	for _, n := range []string{"go1.25.9", "go1.27.0", "go1.26.3", "go1.26.10", "go1.27.1", "go1.24.2"} {
		tg, _ := parseVersionTag(n)
		tags = append(tags, tg)
	}
	lts, current, ok := pickLTS(tags)
	if !ok || lts.name != "go1.26.10" || current.name != "go1.27.1" {
		t.Errorf("pickLTS = %q (current %q) ok=%v, want go1.26.10 (current go1.27.1)", lts.name, current.name, ok)
	}

	one, _ := parseVersionTag("v0.19.1")
	if _, _, ok := pickLTS([]versionTag{one}); ok {
		t.Error("pickLTS with a single release line should fail")
	}
}

func TestGitHubSourceAndPrefix(t *testing.T) {
	spec := &VersionSpec{Strategy: "go-mod", VersionPrefix: "go",
		Fallback: &VersionSpec{Strategy: "github-tags", Repo: "golang/go"}}
	src := githubSource(spec)
	if src == nil || src.Repo != "golang/go" || src.Fallback != nil {
		t.Fatalf("githubSource = %+v", src)
	}
	if p := versionPrefix(spec); p != "go" {
		t.Errorf("versionPrefix = %q", p)
	}
	if githubSource(&VersionSpec{Strategy: "go-mod"}) != nil {
		t.Error("expected no GitHub source")
	}
}

func TestApplyUse(t *testing.T) {
	goInst := &InstallSpec{Strategy: "go-install", Import: "example.com/tool/cmd/tool@latest"}
	v, inst, err := applyUse("1.2.3", nil, goInst)
	if err != nil || v != "v1.2.3" || inst.Import != "example.com/tool/cmd/tool@v1.2.3" {
		t.Errorf("go-install explicit: v=%q import=%q err=%v", v, inst.Import, err)
	}
	if goInst.Import != "example.com/tool/cmd/tool@latest" {
		t.Error("applyUse mutated the recipe's install spec")
	}

	brew := &InstallSpec{Strategy: "brew-install", Package: "go"}
	if _, _, err := applyUse("latest", nil, brew); err != nil {
		t.Errorf("brew latest: %v", err)
	}
	if _, _, err := applyUse("1.26.4", nil, brew); err == nil {
		t.Error("brew explicit version should be rejected")
	}

	spec := &VersionSpec{Strategy: "go-mod", VersionPrefix: "go"}
	tar := &InstallSpec{Strategy: "extract-tar"}
	if v, _, err := applyUse("1.26.3", spec, tar); err != nil || v != "go1.26.3" {
		t.Errorf("explicit golang: v=%q err=%v", v, err)
	}
	if _, _, err := applyUse("latest", spec, tar); err == nil {
		t.Error("latest without a GitHub source should be rejected")
	}
}
