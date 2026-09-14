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

// clogMrmxfPolicy mirrors clog-mrmxf's .clog.yaml, parsed through JSON so the
// string-or-list tags form is exercised.
func clogMrmxfPolicy(t *testing.T) Policy {
	t.Helper()
	var p Policy
	raw := `{"build": ["branch", "tag", "dispatch", "schedule"],
	         "deploy": {"stage": {"branches": ["main", "rc", "dev", "release/*"]},
	                    "prod":  {"tags": "v*", "releases-yaml": "prod", "schedule": true}}}`
	if err := json.Unmarshal([]byte(raw), &p); err != nil {
		t.Fatal(err)
	}
	return p
}

func withPolicy(t *testing.T, p Policy, releaseBuild string) {
	t.Helper()
	savedCfg, savedRel := LoadConfig, ReleaseBuild
	LoadConfig = func() (Config, error) { return Config{Policy: p}, nil }
	ReleaseBuild = func() string { return releaseBuild }
	t.Cleanup(func() { LoadConfig, ReleaseBuild = savedCfg, savedRel })
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

func TestDecide(t *testing.T) {
	withEnvironments(t, nil)
	pol := clogMrmxfPolicy(t)

	tests := []struct {
		name         string
		env          Env
		release      string
		policy       *Policy
		wantEnv      string
		wantBuild    bool
		wantDeploy   bool
		reasonSubstr string
	}{
		{name: "branch main → stage deploy", env: ghEnv("push", "refs/heads/main", nil),
			wantEnv: "stage", wantBuild: true, wantDeploy: true, reasonSubstr: `matches "main"`},
		{name: "feature branch builds, no deploy", env: ghEnv("push", "refs/heads/feature/x", nil),
			wantEnv: "stage", wantBuild: true, wantDeploy: false, reasonSubstr: "not allowed"},
		{name: "* crosses slash", env: ghEnv("push", "refs/heads/release/2026/09", nil),
			wantEnv: "stage", wantBuild: true, wantDeploy: true, reasonSubstr: `"release/*"`},
		{name: "v tag with prod release → prod deploy", env: ghEnv("push", "refs/tags/v1.2.3", nil), release: "prod",
			wantEnv: "prod", wantBuild: true, wantDeploy: true, reasonSubstr: "releases.yaml build is prod"},
		{name: "v tag but releases.yaml says dev", env: ghEnv("push", "refs/tags/v1.2.3", nil), release: "dev",
			wantEnv: "prod", wantBuild: true, wantDeploy: false, reasonSubstr: `want "prod"`},
		{name: "non-v tag", env: ghEnv("push", "refs/tags/nightly", nil), release: "prod",
			wantEnv: "prod", wantBuild: true, wantDeploy: false},
		{name: "pull request never deploys", env: ghEnv("pull_request", "refs/pull/1/merge", nil),
			policy:  &Policy{Deploy: map[string]DeployRule{"dev": {Branches: stringList{"*"}}}},
			wantEnv: "dev", wantBuild: true, wantDeploy: false, reasonSubstr: "never deploy"},
		{name: "pr not in build list", env: ghEnv("pull_request", "refs/pull/1/merge", nil),
			wantEnv: "dev", wantBuild: false, wantDeploy: false, reasonSubstr: "not in ci.policy.build"},
		{name: "dispatch on main", env: ghEnv("workflow_dispatch", "refs/heads/main", nil),
			wantEnv: "stage", wantBuild: true, wantDeploy: true},
		{name: "gitlab schedule → prod when schedule: true", release: "prod",
			env:     fakeEnv(map[string]string{"GITLAB_CI": "true", "CI_PIPELINE_SOURCE": "schedule", "CI_DEFAULT_BRANCH": "main"}, ""),
			wantEnv: "prod", wantBuild: true, wantDeploy: true, reasonSubstr: "schedule: true"},
		{name: "gitlab tag", release: "prod",
			env:     fakeEnv(map[string]string{"GITLAB_CI": "true", "CI_PIPELINE_SOURCE": "push", "CI_COMMIT_REF_NAME": "v2.0.0", "CI_COMMIT_TAG": "v2.0.0"}, ""),
			wantEnv: "prod", wantBuild: true, wantDeploy: true},
		{name: "CLOG_ENV=prod cannot deploy a branch", env: ghEnv("push", "refs/heads/main", map[string]string{"CLOG_ENV": "prod"}), release: "prod",
			wantEnv: "prod", wantBuild: true, wantDeploy: false},
		{name: "not built → not deployed", env: ghEnv("push", "refs/heads/main", nil),
			policy:  &Policy{Build: []Event{EventTag}, Deploy: pol.Deploy},
			wantEnv: "stage", wantBuild: false, wantDeploy: false, reasonSubstr: "nothing to deploy"},
		{name: "actor not allowed: no build, no deploy", env: ghEnv("push", "refs/heads/main", nil),
			policy:  &Policy{Build: pol.Build, Deploy: pol.Deploy, Actors: []string{"CharlottesWeb2"}},
			wantEnv: "stage", wantBuild: false, wantDeploy: false, reasonSubstr: "not in ci.policy.actors"},
		{name: "actor allowed case-insensitively", env: ghEnv("push", "refs/heads/main", nil),
			policy:  &Policy{Build: pol.Build, Deploy: pol.Deploy, Actors: []string{"MRMXF"}},
			wantEnv: "stage", wantBuild: true, wantDeploy: true},
		{name: "no policy builds, never deploys", env: ghEnv("push", "refs/heads/main", nil), policy: &Policy{},
			wantEnv: "stage", wantBuild: true, wantDeploy: false, reasonSubstr: "no ci.policy.deploy.stage"},
		{name: "laptop defaults to dev: no deploy",
			env:     Env{Getenv: func(string) string { return "" }, Git: fakeGit{ref: "main"}},
			wantEnv: "dev", wantBuild: false, wantDeploy: false},
		{name: "laptop CLOG_ENV=stage previews a branch push",
			env:     Env{Getenv: func(k string) string { return map[string]string{"CLOG_ENV": "stage"}[k] }, Git: fakeGit{ref: "main"}},
			wantEnv: "stage", wantBuild: true, wantDeploy: true, reasonSubstr: `branch main matches "main"`},
		{name: "laptop on branch AND tag previews stage as the branch",
			env:     Env{Getenv: func(k string) string { return map[string]string{"CLOG_ENV": "stage"}[k] }, Git: fakeTagGit{fakeGit{ref: "main", onTag: true}, "v3.0.0"}},
			wantEnv: "stage", wantBuild: true, wantDeploy: true, reasonSubstr: "branch main"},
		{name: "laptop CLOG_ENV=prod on a tag previews a tag push", release: "prod",
			env:     Env{Getenv: func(k string) string { return map[string]string{"CLOG_ENV": "prod"}[k] }, Git: fakeTagGit{fakeGit{ref: "main", onTag: true}, "v3.0.0"}},
			wantEnv: "prod", wantBuild: true, wantDeploy: true, reasonSubstr: "tag v3.0.0"},
		{name: "laptop CLOG_ENV=prod off a tag cannot deploy", release: "prod",
			env:     Env{Getenv: func(k string) string { return map[string]string{"CLOG_ENV": "prod"}[k] }, Git: fakeGit{ref: "main"}},
			wantEnv: "prod", wantBuild: true, wantDeploy: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := pol
			if tc.policy != nil {
				p = *tc.policy
			}
			withPolicy(t, p, tc.release)
			d, err := Decide(tc.env)
			if err != nil {
				t.Fatal(err)
			}
			if d.Env != tc.wantEnv || d.Build != tc.wantBuild || d.Deploy != tc.wantDeploy {
				t.Errorf("got env=%s build=%v deploy=%v (%s | %s), want env=%s build=%v deploy=%v",
					d.Env, d.Build, d.Deploy, d.BuildReason, d.DeployReason, tc.wantEnv, tc.wantBuild, tc.wantDeploy)
			}
			if tc.reasonSubstr != "" && !strings.Contains(d.BuildReason+" | "+d.DeployReason, tc.reasonSubstr) {
				t.Errorf("reasons %q / %q do not mention %q", d.BuildReason, d.DeployReason, tc.reasonSubstr)
			}
		})
	}
}

func TestDecisionEnvLines(t *testing.T) {
	d := Decision{Env: "stage", Build: true, Deploy: false}
	if got := d.EnvLines(); got != "clog_env=stage\ndo_build=true\ndo_deploy=false\n" {
		t.Errorf("EnvLines = %q", got)
	}
}

func TestStringListRejectsNumbers(t *testing.T) {
	var s stringList
	if err := json.Unmarshal([]byte(`42`), &s); err == nil {
		t.Error("a number is neither a string nor a list")
	}
}
