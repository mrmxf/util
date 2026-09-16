//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/

package ci

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

const (
	testProject  = "proj-1"
	testDevUUID  = "dev-identity"
	testProdUUID = "prod-identity"
	testJWT      = "header.payload.signature"
	testAccess   = "access-token-xyz"
)

// fakeInfisical serves the OIDC login and secrets endpoints plus a GitHub
// ID-token endpoint, recording what it was asked for.
type fakeInfisical struct {
	srv           *httptest.Server
	loginIdentity string
	fetchEnv      string
	fetchPath     string
	audience      string
}

func newFakeInfisical(t *testing.T) *fakeInfisical {
	t.Helper()
	f := &fakeInfisical{}
	mux := http.NewServeMux()
	mux.HandleFunc("/gh-token", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "bearer gh-request-token" {
			http.Error(w, `{"message":"bad request token"}`, http.StatusUnauthorized)
			return
		}
		f.audience = r.URL.Query().Get("audience")
		json.NewEncoder(w).Encode(map[string]string{"value": testJWT})
	})
	mux.HandleFunc("/api/v1/auth/oidc-auth/login", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)
		if body["jwt"] != testJWT {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"message": "Access denied: OIDC subject not allowed."})
			return
		}
		f.loginIdentity = body["identityId"]
		json.NewEncoder(w).Encode(map[string]any{"accessToken": testAccess, "expiresIn": 900})
	})
	mux.HandleFunc("/api/v4/secrets", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+testAccess {
			http.Error(w, `{"message":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		q := r.URL.Query()
		f.fetchEnv, f.fetchPath = q.Get("environment"), q.Get("secretPath")
		if q.Get("projectId") != testProject || q.Get("includeImports") != "true" {
			http.Error(w, `{"message":"bad query"}`, http.StatusBadRequest)
			return
		}
		w.Write([]byte(`{
			"secrets": [
				{"secretKey": "AWS_ACCESS_KEY_ID", "secretValue": "folder-key"},
				{"secretKey": "SHARED_AND_FOLDER", "secretValue": "folder-wins"}
			],
			"imports": [
				{"secretPath": "/first", "secrets": [
					{"secretKey": "HOOK_SLACK", "secretValue": "first-import"},
					{"secretKey": "SHARED_AND_FOLDER", "secretValue": "import-loses"}
				]},
				{"secretPath": "/second", "secrets": [
					{"secretKey": "HOOK_SLACK", "secretValue": "last-import-wins"},
					{"secretKey": "MULTI", "secretValue": "line one\nline two"}
				]}
			]
		}`))
	})
	f.srv = httptest.NewServer(mux)
	t.Cleanup(f.srv.Close)
	return f
}

func testConfig(domain string) Config {
	return Config{
		Infisical: InfisicalConfig{
			Domain: domain, ProjectID: testProject, Path: "/clog-mrmxf",
			Env: map[string]string{"dev": "dev", "prod": "prod"},
			Identity: map[string]map[string]string{
				"github": {"dev-uuid": testDevUUID, "prod-uuid": testProdUUID},
				"gitlab": {"dev-uuid": testDevUUID, "prod-uuid": testProdUUID},
			},
		},
		Require: map[string]Requirement{
			"deploy": {Env: []string{"AWS_ACCESS_KEY_ID"}, Optional: []string{"HOOK_SLACK"}, Config: []string{"ci.targets.bucket.dev.bucket"}},
		},
		Policy: Policy{Deploy: map[string]DeployRule{
			"dev":  {Branches: stringList{"main"}},
			"prod": {Tags: stringList{"v*"}},
		}},
		Targets: map[string]Target{"bucket": {Kind: KindBucket, Require: []string{"AWS_SECRET_ACCESS_KEY"},
			Dev:  map[string]any{"bucket": "b", "prefix": "clogbin/dev"},
			Prod: map[string]any{"bucket": "b", "prefix": "clogbin/{tag}"}}},
	}
}

func withConfig(t *testing.T, cfg Config, values map[string]any) {
	t.Helper()
	savedCfg, savedVal := LoadConfig, ConfigValue
	LoadConfig = func() (Config, error) { return cfg, nil }
	ConfigValue = func(key string) any { return values[key] }
	t.Cleanup(func() { LoadConfig, ConfigValue = savedCfg, savedVal })
}

func captureLogs(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Cleanup(func() { slog.SetDefault(prev) })
	return &buf
}

func githubPushVars(domain, ref string) map[string]string {
	return map[string]string{
		"GITHUB_ACTIONS":                 "true",
		"GITHUB_EVENT_NAME":              "push",
		"GITHUB_EVENT_PATH":              "/event.json",
		"GITHUB_REPOSITORY":              "mrmxf/clog-mrmxf",
		"GITHUB_REF":                     ref,
		"ACTIONS_ID_TOKEN_REQUEST_URL":   domain + "/gh-token?api-version=2.0",
		"ACTIONS_ID_TOKEN_REQUEST_TOKEN": "gh-request-token",
	}
}

const githubPushEvent = `{"repository": {"html_url": "https://github.com/mrmxf/clog-mrmxf", "default_branch": "main"}, "sender": {"login": "mrmxf"}}`

func TestInfisicalFetchMergesLikeTheCLI(t *testing.T) {
	f := newFakeInfisical(t)
	secrets, err := infisicalFetch(f.srv.URL, testAccess, testProject, "dev", "/clog-mrmxf")
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, s := range secrets {
		if _, dup := got[s.Key]; dup {
			t.Errorf("duplicate key %s", s.Key)
		}
		got[s.Key] = s.Value
	}
	want := map[string]string{
		"AWS_ACCESS_KEY_ID": "folder-key",
		"SHARED_AND_FOLDER": "folder-wins",      // folder beats import
		"HOOK_SLACK":        "last-import-wins", // later import beats earlier
		"MULTI":             "line one\nline two",
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s = %q, want %q", k, got[k], v)
		}
	}
}

func TestInfisicalLoginErrorHidesJWT(t *testing.T) {
	f := newFakeInfisical(t)
	_, err := infisicalLogin(f.srv.URL, testDevUUID, "wrong.jwt.value")
	if err == nil || !strings.Contains(err.Error(), "OIDC subject not allowed") {
		t.Fatalf("want Infisical's message in the error, got %v", err)
	}
	if strings.Contains(err.Error(), "wrong.jwt.value") {
		t.Error("error leaks the JWT")
	}
}

func TestIdentityAndEnvSelection(t *testing.T) {
	inf := testConfig("https://example").Infisical
	for _, tc := range []struct{ clogEnv, wantKey, wantUUID, wantInf string }{
		{"prod", "prod-uuid", testProdUUID, "prod"},
		{"dev", "dev-uuid", testDevUUID, "dev"},
	} {
		key, uuid, err := inf.identityFor(PlatformGitHub, tc.clogEnv)
		if err != nil || key != tc.wantKey || uuid != tc.wantUUID {
			t.Errorf("%s: identity %s/%s err=%v", tc.clogEnv, key, uuid, err)
		}
		if slug, _ := inf.infisicalEnv(tc.clogEnv); slug != tc.wantInf {
			t.Errorf("%s: infisical env %q, want %q", tc.clogEnv, slug, tc.wantInf)
		}
	}
	inf.Identity["github"]["prod-uuid"] = ""
	if _, _, err := inf.identityFor(PlatformGitHub, "prod"); err == nil || !strings.Contains(err.Error(), "ci.infisical.identity.github.prod-uuid") {
		t.Errorf("want error naming the missing key, got %v", err)
	}
	if got := inf.audience(); got != "https://example" {
		t.Errorf("audience defaults to domain, got %q", got)
	}
}

func TestRunGitHubPushUsesDevIdentityAndMasks(t *testing.T) {
	f := newFakeInfisical(t)
	withConfig(t, testConfig(f.srv.URL), nil)
	logs := captureLogs(t)

	env := fakeEnv(githubPushVars(f.srv.URL, "refs/heads/main"), githubPushEvent)
	var out bytes.Buffer
	code, err := Run(env, []string{"clog", "deploy"}, &out, true)
	if err != nil || code != 0 {
		t.Fatalf("Run: code=%d err=%v", code, err)
	}
	if f.loginIdentity != testDevUUID || f.fetchEnv != "dev" || f.fetchPath != "/clog-mrmxf" {
		t.Errorf("branch push should use the dev identity/env: identity=%s env=%s path=%s", f.loginIdentity, f.fetchEnv, f.fetchPath)
	}
	if f.audience != f.srv.URL {
		t.Errorf("GitHub token audience = %q, want the Infisical domain", f.audience)
	}
	masks := out.String()
	for _, v := range []string{"::add-mask::folder-key", "::add-mask::line one", "::add-mask::line two"} {
		if !strings.Contains(masks, v) {
			t.Errorf("missing %q in mask output:\n%s", v, masks)
		}
	}
	for _, secret := range []string{"folder-key", "last-import-wins", testAccess, testJWT} {
		if strings.Contains(logs.String(), secret) {
			t.Errorf("log leaks %q:\n%s", secret, logs.String())
		}
	}
	if !strings.Contains(logs.String(), "mode=dev") {
		t.Errorf("log should name the mode:\n%s", logs.String())
	}
	if !strings.Contains(logs.String(), "AWS_ACCESS_KEY_ID") {
		t.Errorf("log should name the secrets loaded:\n%s", logs.String())
	}
}

func TestRunGitHubTagUsesProdIdentity(t *testing.T) {
	f := newFakeInfisical(t)
	withConfig(t, testConfig(f.srv.URL), nil)
	captureLogs(t)

	env := fakeEnv(githubPushVars(f.srv.URL, "refs/tags/v1.2.3"), githubPushEvent)
	if _, err := Run(env, []string{"true"}, &bytes.Buffer{}, true); err != nil {
		t.Fatal(err)
	}
	if f.loginIdentity != testProdUUID || f.fetchEnv != "prod" {
		t.Errorf("tag push should use prod identity/env: identity=%s env=%s", f.loginIdentity, f.fetchEnv)
	}
}

func TestRunGitLabReadsIDToken(t *testing.T) {
	f := newFakeInfisical(t)
	withConfig(t, testConfig(f.srv.URL), nil)
	captureLogs(t)

	vars := map[string]string{"GITLAB_CI": "true", "CI_PIPELINE_SOURCE": "push", "CI_COMMIT_REF_NAME": "main", "CI_PROJECT_PATH": "mrmxf/clog-mrmxf"}
	if _, err := Run(fakeEnv(vars, ""), []string{"true"}, &bytes.Buffer{}, true); err == nil || !strings.Contains(err.Error(), GitLabIDTokenVar) {
		t.Fatalf("missing id token should name %s, got %v", GitLabIDTokenVar, err)
	}
	vars[GitLabIDTokenVar] = testJWT
	var out bytes.Buffer
	if _, err := Run(fakeEnv(vars, ""), []string{"true"}, &out, true); err != nil {
		t.Fatal(err)
	}
	if f.loginIdentity != testDevUUID {
		t.Errorf("gitlab branch push identity = %s", f.loginIdentity)
	}
	if strings.Contains(out.String(), "::add-mask::") {
		t.Error("GitLab must not get GitHub mask commands")
	}
}

func TestRunPullRequestNeverFetches(t *testing.T) {
	f := newFakeInfisical(t)
	withConfig(t, testConfig(f.srv.URL), nil)
	logs := captureLogs(t)

	vars := map[string]string{"GITHUB_ACTIONS": "true", "GITHUB_EVENT_NAME": "pull_request", "GITHUB_EVENT_PATH": "/event.json", "GITHUB_REPOSITORY": "mrmxf/clog-mrmxf"}
	event := `{"pull_request": {"head": {"sha": "abc", "repo": {"full_name": "fork/clog-mrmxf", "html_url": "https://github.com/fork/clog-mrmxf"}}}, "sender": {"login": "someone"}}`
	if _, err := Run(fakeEnv(vars, event), []string{"true"}, &bytes.Buffer{}, true); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(logs.String(), "without secrets") {
		t.Errorf("expected a warning that the PR runs without secrets:\n%s", logs.String())
	}
	if f.loginIdentity != "" || f.fetchEnv != "" {
		t.Errorf("a pull request must not reach Infisical (identity=%q env=%q)", f.loginIdentity, f.fetchEnv)
	}
}

func TestGitHubJWTNeedsPermission(t *testing.T) {
	_, err := githubJWT(fakeEnv(map[string]string{}, ""), "aud")
	if err == nil || !strings.Contains(err.Error(), "id-token: write") {
		t.Fatalf("want a hint about id-token permission, got %v", err)
	}
}

func TestChildEnv(t *testing.T) {
	captureLogs(t)
	env := childEnv([]string{"PATH=/bin", "AWS_ACCESS_KEY_ID=stale", "CLOG_CI_RUN=0"}, []Secret{{"AWS_ACCESS_KEY_ID", "fresh"}})
	joined := strings.Join(env, "\n")
	if strings.Contains(joined, "stale") || !strings.Contains(joined, "AWS_ACCESS_KEY_ID=fresh") {
		t.Errorf("secret should replace inherited var:\n%s", joined)
	}
	if strings.Count(joined, RunMarkerVar+"=") != 1 || !strings.Contains(joined, RunMarkerVar+"=1") {
		t.Errorf("want exactly one %s=1:\n%s", RunMarkerVar, joined)
	}
	if !strings.Contains(joined, "PATH=/bin") {
		t.Error("unrelated vars must pass through")
	}
}

func TestRunChildExitCode(t *testing.T) {
	code, err := runChild([]string{"sh", "-c", "exit 7"}, os.Environ())
	if err != nil || code != 7 {
		t.Errorf("code=%d err=%v, want 7", code, err)
	}
	if _, err := runChild([]string{"no-such-command-clog-test"}, os.Environ()); err == nil {
		t.Error("missing command should be an error")
	}
}

func TestRequire(t *testing.T) {
	cfg := testConfig("https://example")
	local := func(vars map[string]string) Env {
		e := fakeEnv(vars, "")
		e.Git = fakeGit{ref: "main", slug: "mrmxf/clog-mrmxf"}
		return e
	}

	t.Run("all present", func(t *testing.T) {
		withConfig(t, cfg, map[string]any{"ci.targets.bucket.dev.bucket": "b"})
		captureLogs(t)
		if err := Require(local(map[string]string{"AWS_ACCESS_KEY_ID": "x", "HOOK_SLACK": "y"}), "deploy"); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("missing secret and config", func(t *testing.T) {
		withConfig(t, cfg, map[string]any{"ci.targets.bucket.dev.bucket": ""})
		logs := captureLogs(t)
		err := Require(local(map[string]string{}), "deploy")
		if err == nil || !strings.Contains(err.Error(), "AWS_ACCESS_KEY_ID") || !strings.Contains(err.Error(), "ci.targets.bucket.dev.bucket") {
			t.Fatalf("want both missing names in the error, got %v", err)
		}
		if !strings.Contains(err.Error(), "clog ci run -- clog deploy") || !strings.Contains(err.Error(), "set in .clog.yaml") {
			t.Errorf("want a fix for each kind of missing value: %v", err)
		}
		if !strings.Contains(logs.String(), "level=WARN") || !strings.Contains(logs.String(), "HOOK_SLACK") {
			t.Errorf("optional HOOK_SLACK should warn:\n%s", logs.String())
		}
	})

	t.Run("inside clog ci run points at Infisical", func(t *testing.T) {
		withConfig(t, cfg, map[string]any{"ci.targets.bucket.dev.bucket": "b"})
		captureLogs(t)
		err := Require(local(map[string]string{RunMarkerVar: "1"}), "deploy")
		if err == nil || !strings.Contains(err.Error(), "path=/clog-mrmxf") {
			t.Fatalf("want the Infisical location, got %v", err)
		}
	})

	t.Run("config only does not suggest secrets", func(t *testing.T) {
		withConfig(t, cfg, map[string]any{})
		captureLogs(t)
		err := Require(local(map[string]string{"AWS_ACCESS_KEY_ID": "x", "AWS_SECRET_ACCESS_KEY": "y"}), "deploy")
		if err == nil || strings.Contains(err.Error(), "clog ci run") || !strings.Contains(err.Error(), ".clog.yaml") {
			t.Fatalf("missing config should point at .clog.yaml only, got %v", err)
		}
	})

	t.Run("verb without requirements passes", func(t *testing.T) {
		withConfig(t, cfg, nil)
		if err := Require(local(map[string]string{}), "build"); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("the target adds its own secrets", func(t *testing.T) {
		withConfig(t, cfg, map[string]any{"ci.targets.bucket.dev.bucket": "b"})
		captureLogs(t)
		env := local(map[string]string{"AWS_ACCESS_KEY_ID": "x", TargetVar: "bucket"})
		err := Require(env, "deploy")
		if err == nil || !strings.Contains(err.Error(), "AWS_SECRET_ACCESS_KEY") {
			t.Fatalf("want the target's require list checked, got %v", err)
		}
	})
}
