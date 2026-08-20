//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/

package ci

import (
	"strings"
	"testing"
)

// withEnvironments swaps the Environments hook for a fixture and restores it.
func withEnvironments(t *testing.T, table map[string]map[string]any) {
	t.Helper()
	saved := Environments
	Environments = func() map[string]map[string]any { return table }
	t.Cleanup(func() { Environments = saved })
}

func sampleTable() map[string]map[string]any {
	return map[string]map[string]any{
		"dev": {
			"base-url":   "http://localhost:1313/",
			"hugo-flags": "--buildDrafts --buildFuture --buildExpired",
			"image-tags": []any{},
		},
		"stage": {
			"base-url":   "https://staging.mrmxf.com/",
			"hugo-flags": "--buildFuture",
			"image-tags": []any{"latest-stage", "2.2.11-stage"},
		},
		"prod": {
			"base-url":   "https://mrmxf.com/",
			"hugo-flags": "",
			"image-tags": []any{"latest", "2.2.11"},
		},
	}
}

// --- the mapping -----------------------------------------------------------

func TestEnvName(t *testing.T) {
	tests := []struct {
		name string
		in   Resolution
		want string
	}{
		{"laptop", Resolution{Verb: VerbLocal}, EnvDev},
		{"laptop sitting on a tag", Resolution{Verb: VerbLocal, IsTag: true}, EnvDev},
		{"pull request", Resolution{Verb: VerbPR}, EnvDev},
		{"branch push", Resolution{Verb: VerbPush}, EnvStage},
		{"tag push", Resolution{Verb: VerbPush, IsTag: true}, EnvProd},
		{"schedule", Resolution{Verb: VerbSchedule, IsProduction: true}, EnvProd},
		{"manual dispatch", Resolution{Verb: VerbDispatch}, EnvStage},
		{"dispatch on a tag", Resolution{Verb: VerbDispatch, IsTag: true}, EnvProd},

		// The case the ordering exists for: a PR must not reach a deployment
		// target even when the event carries production-looking flags.
		{"PR on a tag is still dev", Resolution{Verb: VerbPR, IsTag: true}, EnvDev},
		{"PR flagged production is still dev", Resolution{Verb: VerbPR, IsProduction: true}, EnvDev},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := EnvName(tc.in); got != tc.want {
				t.Errorf("EnvName(%+v) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// --- end-to-end through the resolver ---------------------------------------

func TestResolveEnvName(t *testing.T) {
	withEnvironments(t, sampleTable())

	tests := []struct {
		name string
		vars map[string]string
		want string
	}{
		{
			name: "github branch push builds stage",
			vars: map[string]string{
				"GITHUB_ACTIONS":    "true",
				"GITHUB_EVENT_NAME": "push",
				"GITHUB_REF":        "refs/heads/main",
			},
			want: EnvStage,
		},
		{
			name: "github tag push builds prod",
			vars: map[string]string{
				"GITHUB_ACTIONS":    "true",
				"GITHUB_EVENT_NAME": "push",
				"GITHUB_REF":        "refs/tags/v2.2.12",
				"GITHUB_REF_TYPE":   "tag",
			},
			want: EnvProd,
		},
		{
			name: "github schedule builds prod",
			vars: map[string]string{
				"GITHUB_ACTIONS":    "true",
				"GITHUB_EVENT_NAME": "schedule",
			},
			want: EnvProd,
		},
		{
			name: "github pull_request builds dev",
			vars: map[string]string{
				"GITHUB_ACTIONS":    "true",
				"GITHUB_EVENT_NAME": "pull_request",
			},
			want: EnvDev,
		},
		// Platform parity: the same intent on GitLab must give the same answer,
		// which is the whole point of resolving before deciding.
		{
			name: "gitlab branch push builds stage",
			vars: map[string]string{
				"GITLAB_CI":          "true",
				"CI_PIPELINE_SOURCE": "push",
				"CI_COMMIT_REF_NAME": "main",
			},
			want: EnvStage,
		},
		{
			name: "gitlab tag push builds prod",
			vars: map[string]string{
				"GITLAB_CI":          "true",
				"CI_PIPELINE_SOURCE": "push",
				"CI_COMMIT_TAG":      "v2.2.12",
			},
			want: EnvProd,
		},
		{
			name: "gitlab schedule builds prod",
			vars: map[string]string{
				"GITLAB_CI":          "true",
				"CI_PIPELINE_SOURCE": "schedule",
			},
			want: EnvProd,
		},
		{
			name: "gitlab merge request builds dev",
			vars: map[string]string{
				"GITLAB_CI":          "true",
				"CI_PIPELINE_SOURCE": "merge_request_event",
			},
			want: EnvDev,
		},
		{
			name: "CLOG_ENV overrides the mapping",
			vars: map[string]string{
				"GITHUB_ACTIONS":    "true",
				"GITHUB_EVENT_NAME": "push",
				"GITHUB_REF":        "refs/heads/main",
				EnvOverrideVar:      "prod",
			},
			want: EnvProd,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ResolveEnvName(fakeEnv(tc.vars, "{}"))
			if err != nil {
				t.Fatalf("ResolveEnvName: unexpected error %v", err)
			}
			if got != tc.want {
				t.Errorf("ResolveEnvName = %q, want %q", got, tc.want)
			}
		})
	}
}

// A typo in CLOG_ENV must fail the build rather than silently fall back — the
// failure mode it guards against is publishing the wrong site.
func TestResolveEnvNameRejectsUnknownOverride(t *testing.T) {
	withEnvironments(t, sampleTable())

	_, err := ResolveEnvName(fakeEnv(map[string]string{EnvOverrideVar: "staging"}, "{}"))
	if err == nil {
		t.Fatal("expected an error for CLOG_ENV=staging, got nil")
	}
	for _, want := range []string{"staging", "dev", "prod", "stage"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q should mention %q", err, want)
		}
	}
}

// With no environments table configured the override cannot be validated, so it
// is accepted: a repo that has not adopted the table yet must not be blocked.
func TestResolveEnvNameOverrideWithoutTable(t *testing.T) {
	withEnvironments(t, nil)

	got, err := ResolveEnvName(fakeEnv(map[string]string{EnvOverrideVar: "anything"}, "{}"))
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	if got != "anything" {
		t.Errorf("got %q, want %q", got, "anything")
	}
}

// --- reading the table -----------------------------------------------------

func TestEnvGet(t *testing.T) {
	withEnvironments(t, sampleTable())

	tests := []struct {
		name    string
		env     string
		key     string
		want    string
		wantErr string
	}{
		{name: "scalar", env: "prod", key: "base-url", want: "https://mrmxf.com/"},
		{name: "flags with spaces survive intact", env: "dev", key: "hugo-flags",
			want: "--buildDrafts --buildFuture --buildExpired"},
		{name: "list is one per line", env: "stage", key: "image-tags",
			want: "latest-stage\n2.2.11-stage"},
		{name: "empty list is empty, not an error", env: "dev", key: "image-tags", want: ""},
		{name: "empty scalar is empty, not an error", env: "prod", key: "hugo-flags", want: ""},
		{name: "unknown key names the ones that exist", env: "prod", key: "baseurl",
			wantErr: "base-url"},
		{name: "unknown environment names the ones that exist", env: "production", key: "base-url",
			wantErr: "not configured"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := EnvGet(tc.env, tc.key)
			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("expected an error mentioning %q, got %q", tc.wantErr, got)
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Errorf("error %q should mention %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error %v", err)
			}
			if got != tc.want {
				t.Errorf("EnvGet(%q, %q) = %q, want %q", tc.env, tc.key, got, tc.want)
			}
		})
	}
}

// A missing table is a configuration error worth naming, because the symptom
// otherwise is an empty --baseURL and a subtly wrong site.
func TestEnvRowWithoutTable(t *testing.T) {
	withEnvironments(t, nil)

	_, err := EnvRow(EnvProd)
	if err == nil {
		t.Fatal("expected an error when no environments table is configured")
	}
	if !strings.Contains(err.Error(), EnvironmentsKey) {
		t.Errorf("error %q should mention the %q key", err, EnvironmentsKey)
	}
}

func TestRenderValue(t *testing.T) {
	tests := []struct {
		name string
		in   any
		want string
	}{
		{"nil", nil, ""},
		{"string", "hello", "hello"},
		{"int", 8080, "8080"},
		{"bool", true, "true"},
		{"any slice", []any{"a", "b"}, "a\nb"},
		{"string slice", []string{"a", "b"}, "a\nb"},
		{"empty slice", []any{}, ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := renderValue(tc.in); got != tc.want {
				t.Errorf("renderValue(%v) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
