//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/

package ci

import (
	"encoding/json"
	"strings"
	"testing"
)

type fakeTagGit struct {
	fakeGit
	tag string
}

func (f fakeTagGit) ExactTag() string { return f.tag }

// clogMrmxfConfig mirrors clog-mrmxf's .clog.yaml (policy + one bucket target),
// parsed through JSON so the string-or-list and target shapes are exercised.
func clogMrmxfConfig(t *testing.T) Config {
	t.Helper()
	var cfg Config
	raw := `{
	  "policy": {
	    "build": ["branch", "tag", "dispatch", "schedule"],
	    "deploy": {
	      "dev":  {"branches": ["main", "rc", "dev", "release/*"]},
	      "prod": {"tags": "v*", "releases-yaml": "prod", "schedule": true}
	    }
	  },
	  "targets": {
	    "bucket": {
	      "kind": "bucket",
	      "require": ["AWS_ACCESS_KEY_ID"],
	      "dev":  {"bucket": "b", "prefix": "clogbin/dev",     "installer": "clogdev"},
	      "prod": {"bucket": "b", "prefix": "clogbin/{tag}",   "installer": "clog"}
	    }
	  }}`
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		t.Fatal(err)
	}
	return cfg
}

func withCfg(t *testing.T, cfg Config) {
	t.Helper()
	savedCfg, savedVer := LoadConfig, ReleaseVersion
	LoadConfig = func() (Config, error) { return cfg, nil }
	ReleaseVersion = func() string { return "v9.9.9" }
	t.Cleanup(func() { LoadConfig, ReleaseVersion = savedCfg, savedVer })
}

func ghEnv(event, ref string, extra map[string]string) Env {
	vars := map[string]string{
		"GITHUB_ACTIONS": "true", "GITHUB_EVENT_NAME": event, "GITHUB_EVENT_PATH": "/event.json",
		"GITHUB_REPOSITORY": "mrmxf/clog-mrmxf", "GITHUB_REF": ref,
	}
	for k, v := range extra {
		vars[k] = v
	}
	payload := githubPushEvent
	if event == "pull_request" {
		payload = `{"pull_request": {"head": {"sha": "abc", "repo": {"full_name": "fork/x", "html_url": "https://github.com/fork/x"}}}, "sender": {"login": "someone"}}`
	}
	return fakeEnv(vars, payload)
}

func laptop(vars map[string]string, git GitResolver) Env {
	return Env{Getenv: func(k string) string { return vars[k] }, Git: git}
}

// TestDecideModes is the §1.1 table of the migration plan: which event, in
// which mode, deploying or not.
func TestDecideModes(t *testing.T) {
	base := clogMrmxfConfig(t)

	tests := []struct {
		name         string
		env          Env
		cfg          *Config
		wantMode     string
		wantBuild    bool
		wantDeploy   bool
		reasonSubstr string
	}{
		{name: "branch push is dev", env: ghEnv("push", "refs/heads/main", nil),
			wantMode: "dev", wantBuild: true, wantDeploy: true, reasonSubstr: `branch main matches "main"`},
		{name: "feature branch builds, does not deploy", env: ghEnv("push", "refs/heads/feature/x", nil),
			wantMode: "dev", wantBuild: true, wantDeploy: false, reasonSubstr: "not allowed"},
		{name: "dispatch is dev", env: ghEnv("workflow_dispatch", "refs/heads/rc", nil),
			wantMode: "dev", wantBuild: true, wantDeploy: true},
		{name: "release tag is prod (releases.yaml is not consulted)", env: ghEnv("push", "refs/tags/v1.2.3", nil),
			wantMode: "prod", wantBuild: true, wantDeploy: true, reasonSubstr: "is a release tag"},
		{name: "pre-release tag matching v* is dev", env: ghEnv("push", "refs/tags/v1.2.3-rc1", nil),
			wantMode: "dev", wantBuild: true, wantDeploy: false, reasonSubstr: "not a release tag"},
		{name: "dev-suffixed tag is dev", env: ghEnv("push", "refs/tags/v3.5.0-dev", nil),
			wantMode: "dev", wantBuild: true, wantDeploy: false, reasonSubstr: "not a release tag"},
		{name: "non-v tag is dev", env: ghEnv("push", "refs/tags/nightly", nil),
			wantMode: "dev", wantBuild: true, wantDeploy: false},
		{name: "pull request is dev and never deploys", env: ghEnv("pull_request", "refs/pull/1/merge", nil),
			wantMode: "dev", wantBuild: false, wantDeploy: false, reasonSubstr: "never deploy"},
		{name: "pull request with pr in the build list still never deploys", env: ghEnv("pull_request", "refs/pull/1/merge", nil),
			cfg: &Config{Policy: Policy{Build: []Event{EventPR}, Deploy: map[string]DeployRule{"dev": {Branches: stringList{"*"}}}},
				Targets: map[string]Target{"bucket": {Kind: KindBucket, Dev: map[string]any{"bucket": "b"}}}},
			wantMode: "dev", wantBuild: true, wantDeploy: false, reasonSubstr: "never deploy"},
		{name: "schedule is prod when allowed",
			env:      fakeEnv(map[string]string{"GITLAB_CI": "true", "CI_PIPELINE_SOURCE": "schedule", "CI_DEFAULT_BRANCH": "main"}, ""),
			wantMode: "prod", wantBuild: true, wantDeploy: true, reasonSubstr: "scheduled"},
		{name: "gitlab tag is prod",
			env:      fakeEnv(map[string]string{"GITLAB_CI": "true", "CI_PIPELINE_SOURCE": "push", "CI_COMMIT_REF_NAME": "v2.0.0", "CI_COMMIT_TAG": "v2.0.0"}, ""),
			wantMode: "prod", wantBuild: true, wantDeploy: true},
		{name: "CLOG_MODE=prod on a branch: prod mode, but the branch cannot deploy prod",
			env:      ghEnv("push", "refs/heads/main", map[string]string{"CLOG_MODE": "prod"}),
			wantMode: "prod", wantBuild: true, wantDeploy: false, reasonSubstr: "is not allowed by ci.policy.deploy.prod"},
		{name: "laptop is dev and does not deploy (local is not in the build list)", env: laptop(nil, fakeGit{ref: "main"}),
			wantMode: "dev", wantBuild: false, wantDeploy: false},
		{name: "laptop CLOG_MODE=dev previews the branch push",
			env:      laptop(map[string]string{"CLOG_MODE": "dev"}, fakeGit{ref: "main"}),
			wantMode: "dev", wantBuild: true, wantDeploy: true, reasonSubstr: "branch main"},
		{name: "laptop CLOG_MODE=prod on a tag previews the tag push",
			env:      laptop(map[string]string{"CLOG_MODE": "prod"}, fakeTagGit{fakeGit{ref: "main", onTag: true}, "v3.0.0"}),
			wantMode: "prod", wantBuild: true, wantDeploy: true, reasonSubstr: "tag v3.0.0"},
		{name: "actor not allowed", env: ghEnv("push", "refs/heads/main", nil),
			cfg:      &Config{Policy: Policy{Build: base.Policy.Build, Deploy: base.Policy.Deploy, Actors: []string{"someone-else"}}, Targets: base.Targets},
			wantMode: "dev", wantBuild: false, wantDeploy: false, reasonSubstr: "ci.policy.actors"},
		{name: "no target for the mode = no deploy", env: ghEnv("push", "refs/heads/main", nil),
			cfg: &Config{Policy: base.Policy, Targets: map[string]Target{
				"pages": {Kind: KindCloudflarePage, Modes: []string{"prod"}, Prod: map[string]any{"project": "p"}}}},
			wantMode: "dev", wantBuild: true, wantDeploy: false, reasonSubstr: "no ci.targets deploys in dev mode"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := base
			if tc.cfg != nil {
				cfg = *tc.cfg
			}
			withCfg(t, cfg)
			d, err := Decide(tc.env)
			if err != nil {
				t.Fatal(err)
			}
			if d.Mode != tc.wantMode || d.Build != tc.wantBuild || d.Deploy != tc.wantDeploy {
				t.Errorf("got mode=%s build=%v deploy=%v (%s | %s | %s), want mode=%s build=%v deploy=%v",
					d.Mode, d.Build, d.Deploy, d.ModeReason, d.BuildReason, d.DeployReason, tc.wantMode, tc.wantBuild, tc.wantDeploy)
			}
			if tc.reasonSubstr != "" {
				all := d.ModeReason + " | " + d.BuildReason + " | " + d.DeployReason
				if !strings.Contains(all, tc.reasonSubstr) {
					t.Errorf("reasons %q do not mention %q", all, tc.reasonSubstr)
				}
			}
		})
	}
}

func TestModeOverrideRejectsStage(t *testing.T) {
	withCfg(t, clogMrmxfConfig(t))
	for _, v := range []string{"CLOG_MODE", "CLOG_ENV"} {
		_, err := Decide(ghEnv("push", "refs/heads/main", map[string]string{v: "stage"}))
		if err == nil || !strings.Contains(err.Error(), "staging was removed") {
			t.Errorf("%s=stage should fail loudly, got %v", v, err)
		}
	}
	// the legacy name still works for a real mode
	d, err := Decide(ghEnv("push", "refs/heads/main", map[string]string{"CLOG_ENV": "prod"}))
	if err != nil || d.Mode != ModeProd {
		t.Errorf("CLOG_ENV=prod: mode=%q err=%v", d.Mode, err)
	}
}

func TestDecisionEnvLines(t *testing.T) {
	withCfg(t, clogMrmxfConfig(t))
	d, err := Decide(ghEnv("push", "refs/tags/v1.2.3", nil))
	if err != nil {
		t.Fatal(err)
	}
	want := "build_mode=prod\ndeploy_mode=prod\ndo_build=true\ndo_deploy=true\ndeploy_targets=bucket\n"
	if got := d.EnvLines(); got != want {
		t.Errorf("EnvLines =\n%q\nwant\n%q", got, want)
	}

	d2, err := Decide(ghEnv("push", "refs/heads/feature/x", nil))
	if err != nil {
		t.Fatal(err)
	}
	if got := d2.EnvLines(); !strings.Contains(got, "deploy_mode=\n") || !strings.Contains(got, "deploy_targets=\n") {
		t.Errorf("a non-deploying run should leave deploy_mode and deploy_targets empty:\n%q", got)
	}
}

func TestStringListRejectsNumbers(t *testing.T) {
	var s stringList
	if err := json.Unmarshal([]byte(`42`), &s); err == nil {
		t.Error("a number is neither a string nor a list")
	}
}

// A manual run's ref is the branch it was launched from, so it can never match
// a tag glob - even when the job then checks out the release tag. dispatch:true
// is what lets "republish the current release" be one click.
func TestDeployDispatchRule(t *testing.T) {
	prev := LoadConfig
	defer func() { LoadConfig = prev }()

	withRule := func(r DeployRule) {
		LoadConfig = func() (Config, error) {
			return Config{
				Policy:  Policy{Build: []Event{EventDispatch}, Deploy: map[string]DeployRule{ModeProd: r}},
				Targets: map[string]Target{"pages": {Kind: KindGitHubPages, Modes: []string{ModeProd}, Prod: map[string]any{"dir": "kodata"}}},
			}, nil
		}
	}
	env := ghEnv("workflow_dispatch", "refs/heads/main", map[string]string{"CLOG_MODE": "prod"})

	withRule(DeployRule{Tags: stringList{"v*"}})
	d, err := Decide(env)
	if err != nil {
		t.Fatal(err)
	}
	if d.Deploy {
		t.Errorf("a tags-only rule must not deploy a manual run: %s", d.DeployReason)
	}

	withRule(DeployRule{Tags: stringList{"v*"}, Dispatch: true})
	d, err = Decide(env)
	if err != nil {
		t.Fatal(err)
	}
	if !d.Deploy {
		t.Errorf("dispatch: true should allow a manual deploy: %s", d.DeployReason)
	}
	if len(d.Targets) != 1 || d.Targets[0] != "pages" {
		t.Errorf("targets = %v, want [pages]", d.Targets)
	}
}
