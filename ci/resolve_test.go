//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/

package ci

import (
	"strings"
	"testing"
)

// fakeEnv builds an Env from a map, with an optional GitHub event JSON payload
// served from whatever path GITHUB_EVENT_PATH points at.
func fakeEnv(vars map[string]string, eventJSON string) Env {
	return Env{
		Getenv: func(k string) string { return vars[k] },
		ReadFile: func(path string) ([]byte, error) {
			if path == vars["GITHUB_EVENT_PATH"] {
				return []byte(eventJSON), nil
			}
			return nil, errNotFound
		},
	}
}

type errString string

func (e errString) Error() string { return string(e) }

const errNotFound = errString("not found")

// --- GitHub ----------------------------------------------------------------

func TestResolveGitHub(t *testing.T) {
	const prEvent = `{
		"pull_request": {"head": {"sha": "deadbeef",
			"repo": {"full_name": "fork/clog", "html_url": "https://github.com/fork/clog"}}},
		"sender": {"login": "contributor"}
	}`
	const pushEvent = `{
		"repository": {"html_url": "https://github.com/mrmxf/clog", "default_branch": "main"},
		"sender": {"login": "mrmxf"}
	}`

	tests := []struct {
		name  string
		vars  map[string]string
		event string
		want  Resolution
	}{
		{
			name: "pull_request_target",
			vars: map[string]string{
				"GITHUB_ACTIONS":    "true",
				"GITHUB_EVENT_NAME": "pull_request_target",
				"GITHUB_EVENT_PATH": "/event.json",
				"GITHUB_REPOSITORY": "mrmxf/clog",
				"GITHUB_ACTOR":      "mrmxf",
			},
			event: prEvent,
			want: Resolution{
				CI: PlatformGitHub, Verb: VerbPR, Ref: "deadbeef", Repo: "fork/clog",
				URL: "https://github.com/fork/clog", Depth: 1, Actor: "contributor",
			},
		},
		{
			name: "push",
			vars: map[string]string{
				"GITHUB_ACTIONS":    "true",
				"GITHUB_EVENT_NAME": "push",
				"GITHUB_EVENT_PATH": "/event.json",
				"GITHUB_REPOSITORY": "mrmxf/clog",
				"GITHUB_REF":        "refs/heads/main",
				"GITHUB_ACTOR":      "mrmxf",
			},
			event: pushEvent,
			want: Resolution{
				CI: PlatformGitHub, Verb: VerbPush, Ref: "refs/heads/main", Repo: "mrmxf/clog",
				URL: "https://github.com/mrmxf/clog", Depth: 1, Actor: "mrmxf",
			},
		},
		{
			name: "workflow_dispatch",
			vars: map[string]string{
				"GITHUB_ACTIONS":    "true",
				"GITHUB_EVENT_NAME": "workflow_dispatch",
				"GITHUB_EVENT_PATH": "/event.json",
				"GITHUB_REPOSITORY": "mrmxf/clog",
				"GITHUB_REF":        "refs/heads/main",
			},
			event: pushEvent,
			want: Resolution{
				CI: PlatformGitHub, Verb: VerbDispatch, Ref: "refs/heads/main", Repo: "mrmxf/clog",
				URL: "https://github.com/mrmxf/clog", Depth: 1, Actor: "mrmxf",
			},
		},
		{
			name: "schedule",
			vars: map[string]string{
				"GITHUB_ACTIONS":    "true",
				"GITHUB_EVENT_NAME": "schedule",
				"GITHUB_EVENT_PATH": "/event.json",
				"GITHUB_REPOSITORY": "mrmxf/clog",
				"GITHUB_ACTOR":      "mrmxf",
			},
			event: pushEvent,
			want: Resolution{
				CI: PlatformGitHub, Verb: VerbSchedule, Ref: "main", Repo: "mrmxf/clog",
				URL: "https://github.com/mrmxf/clog", Depth: 0, Actor: "mrmxf", IsProduction: true,
			},
		},
		{
			name: "push tag",
			vars: map[string]string{
				"GITHUB_ACTIONS":    "true",
				"GITHUB_EVENT_NAME": "push",
				"GITHUB_EVENT_PATH": "/event.json",
				"GITHUB_REPOSITORY": "mrmxf/clog",
				"GITHUB_REF":        "refs/tags/v1.2.3",
				"GITHUB_REF_TYPE":   "tag",
			},
			event: pushEvent,
			want: Resolution{
				CI: PlatformGitHub, Verb: VerbPush, Ref: "refs/tags/v1.2.3", Repo: "mrmxf/clog",
				URL: "https://github.com/mrmxf/clog", Depth: 1, Actor: "mrmxf", IsTag: true,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Resolve(fakeEnv(tc.vars, tc.event))
			if err != nil {
				t.Fatalf("Resolve: %v", err)
			}
			assertResolution(t, got, tc.want)
		})
	}
}

// --- GitLab ----------------------------------------------------------------

func TestResolveGitLab(t *testing.T) {
	tests := []struct {
		name string
		vars map[string]string
		want Resolution
	}{
		{
			name: "merge_request_event",
			vars: map[string]string{
				"GITLAB_CI":                            "true",
				"CI_PIPELINE_SOURCE":                   "merge_request_event",
				"CI_PROJECT_PATH":                      "mrmxf/clog",
				"CI_PROJECT_URL":                       "https://gitlab.com/mrmxf/clog",
				"CI_COMMIT_SHA":                        "cafe",
				"CI_MERGE_REQUEST_SOURCE_BRANCH_SHA":   "babe",
				"CI_MERGE_REQUEST_SOURCE_PROJECT_PATH": "fork/clog",
				"CI_MERGE_REQUEST_SOURCE_PROJECT_URL":  "https://gitlab.com/fork/clog",
				"GITLAB_USER_LOGIN":                    "contributor",
			},
			want: Resolution{
				CI: PlatformGitLab, Verb: VerbPR, Ref: "babe", Repo: "fork/clog",
				URL: "https://gitlab.com/fork/clog", Depth: 1, Actor: "contributor",
			},
		},
		{
			name: "push",
			vars: map[string]string{
				"GITLAB_CI":          "true",
				"CI_PIPELINE_SOURCE": "push",
				"CI_PROJECT_PATH":    "mrmxf/clog",
				"CI_PROJECT_URL":     "https://gitlab.com/mrmxf/clog",
				"CI_COMMIT_REF_NAME": "main",
				"GITLAB_USER_LOGIN":  "mrmxf",
			},
			want: Resolution{
				CI: PlatformGitLab, Verb: VerbPush, Ref: "main", Repo: "mrmxf/clog",
				URL: "https://gitlab.com/mrmxf/clog", Depth: 1, Actor: "mrmxf",
			},
		},
		{
			name: "schedule",
			vars: map[string]string{
				"GITLAB_CI":          "true",
				"CI_PIPELINE_SOURCE": "schedule",
				"CI_PROJECT_PATH":    "mrmxf/clog",
				"CI_PROJECT_URL":     "https://gitlab.com/mrmxf/clog",
				"CI_DEFAULT_BRANCH":  "main",
				"GITLAB_USER_LOGIN":  "mrmxf",
			},
			want: Resolution{
				CI: PlatformGitLab, Verb: VerbSchedule, Ref: "main", Repo: "mrmxf/clog",
				URL: "https://gitlab.com/mrmxf/clog", Depth: 0, Actor: "mrmxf", IsProduction: true,
			},
		},
		{
			name: "web dispatch",
			vars: map[string]string{
				"GITLAB_CI":          "true",
				"CI_PIPELINE_SOURCE": "web",
				"CI_PROJECT_PATH":    "mrmxf/clog",
				"CI_PROJECT_URL":     "https://gitlab.com/mrmxf/clog",
				"CI_COMMIT_REF_NAME": "main",
				"GITLAB_USER_LOGIN":  "mrmxf",
			},
			want: Resolution{
				CI: PlatformGitLab, Verb: VerbDispatch, Ref: "main", Repo: "mrmxf/clog",
				URL: "https://gitlab.com/mrmxf/clog", Depth: 1, Actor: "mrmxf",
			},
		},
		{
			name: "push tag",
			vars: map[string]string{
				"GITLAB_CI":          "true",
				"CI_PIPELINE_SOURCE": "push",
				"CI_PROJECT_PATH":    "mrmxf/clog",
				"CI_PROJECT_URL":     "https://gitlab.com/mrmxf/clog",
				"CI_COMMIT_REF_NAME": "v1.2.3",
				"CI_COMMIT_TAG":      "v1.2.3",
				"GITLAB_USER_LOGIN":  "mrmxf",
			},
			want: Resolution{
				CI: PlatformGitLab, Verb: VerbPush, Ref: "v1.2.3", Repo: "mrmxf/clog",
				URL: "https://gitlab.com/mrmxf/clog", Depth: 1, Actor: "mrmxf", IsTag: true,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Resolve(fakeEnv(tc.vars, ""))
			if err != nil {
				t.Fatalf("Resolve: %v", err)
			}
			assertResolution(t, got, tc.want)
		})
	}
}

// --- Local -----------------------------------------------------------------

type fakeGit struct {
	ref, slug, url, user string
	onTag                bool
}

func (f fakeGit) CurrentRef() string { return f.ref }
func (f fakeGit) RepoSlug() string   { return f.slug }
func (f fakeGit) RemoteURL() string  { return f.url }
func (f fakeGit) UserName() string   { return f.user }
func (f fakeGit) OnExactTag() bool   { return f.onTag }

func TestResolveLocal(t *testing.T) {
	env := Env{
		Getenv:   func(k string) string { return "" }, // no CI vars
		ReadFile: func(string) ([]byte, error) { return nil, errNotFound },
		Git: fakeGit{
			ref: "feature/x", slug: "mrmxf/clog",
			url: "git@github.com:mrmxf/clog.git", user: "Bruce", onTag: false,
		},
	}
	got, err := Resolve(env)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	assertResolution(t, got, Resolution{
		CI: PlatformLocal, Verb: VerbLocal, Ref: "feature/x", Repo: "mrmxf/clog",
		URL: "git@github.com:mrmxf/clog.git", Depth: 1, Actor: "Bruce",
	})
}

func TestSlugFromRemote(t *testing.T) {
	cases := map[string]string{
		"git@github.com:mrmxf/clog.git":       "mrmxf/clog",
		"https://github.com/mrmxf/clog.git":   "mrmxf/clog",
		"https://gitlab.com/mrmxf/clog":       "mrmxf/clog",
		"ssh://git@gitlab.com/mrmxf/clog.git": "mrmxf/clog",
		"https://github.com/mrmxf/clog/":      "mrmxf/clog",
		"":                                    "",
	}
	for in, want := range cases {
		if got := slugFromRemote(in); got != want {
			t.Errorf("slugFromRemote(%q) = %q, want %q", in, got, want)
		}
	}
}

// --- env output ------------------------------------------------------------

func TestEnvLines(t *testing.T) {
	r := Resolution{
		CI: PlatformGitHub, Verb: VerbSchedule, Ref: "main", Repo: "mrmxf/clog",
		URL: "https://github.com/mrmxf/clog", Depth: 0, Actor: "mrmxf", IsProduction: true,
	}
	out := r.EnvLines()
	for _, want := range []string{
		"ci=github\n", "verb=SCHEDULE\n", "ref=main\n", "repo=mrmxf/clog\n",
		"depth=0\n", "actor=mrmxf\n", "is_production=true\n", "is_tag=false\n",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("EnvLines() missing %q\ngot:\n%s", want, out)
		}
	}
}

func assertResolution(t *testing.T, got, want Resolution) {
	t.Helper()
	if got != want {
		t.Errorf("resolution mismatch:\n got  %+v\n want %+v", got, want)
	}
}
