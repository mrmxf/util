//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/

package ci

import (
	"encoding/json"
	"os"
	"strings"
)

// Env is the injectable environment the resolver reads. Tests supply fakes;
// production uses DefaultEnv().
type Env struct {
	// Getenv reads an environment variable (os.Getenv in production).
	Getenv func(string) string
	// ReadFile reads a file by path (os.ReadFile in production). Used for the
	// GitHub event payload at $GITHUB_EVENT_PATH.
	ReadFile func(string) ([]byte, error)
	// Git resolves facts from the local working copy in local mode. Injectable
	// so local-mode resolution is testable without a real repo.
	Git GitResolver
}

// DefaultEnv returns an Env backed by the real OS and git.
func DefaultEnv() Env {
	return Env{
		Getenv:   os.Getenv,
		ReadFile: os.ReadFile,
		Git:      execGit{},
	}
}

// Resolve detects the active CI platform and returns the normalized Resolution.
func Resolve(env Env) (Resolution, error) {
	switch {
	case env.Getenv("GITHUB_ACTIONS") == "true":
		return resolveGitHub(env)
	case env.Getenv("GITLAB_CI") == "true":
		return resolveGitLab(env), nil
	default:
		return resolveLocal(env)
	}
}

// ghEvent is the subset of the GitHub event payload the resolver consumes.
type ghEvent struct {
	PullRequest struct {
		Head struct {
			Sha  string `json:"sha"`
			Repo struct {
				FullName string `json:"full_name"`
				HTMLURL  string `json:"html_url"`
			} `json:"repo"`
		} `json:"head"`
	} `json:"pull_request"`
	Repository struct {
		HTMLURL       string `json:"html_url"`
		DefaultBranch string `json:"default_branch"`
	} `json:"repository"`
	Sender struct {
		Login string `json:"login"`
	} `json:"sender"`
}

// resolveGitHub mirrors the legacy build-golang `case "$GITHUB_EVENT_NAME"`
// block, reading the event payload at $GITHUB_EVENT_PATH for PR details.
func resolveGitHub(env Env) (Resolution, error) {
	get := env.Getenv
	r := Resolution{
		CI:    PlatformGitHub,
		Depth: 1,
		Repo:  get("GITHUB_REPOSITORY"),
		Ref:   get("GITHUB_REF"),
		Actor: get("GITHUB_ACTOR"),
		IsTag: get("GITHUB_REF_TYPE") == "tag" || strings.HasPrefix(get("GITHUB_REF"), "refs/tags/"),
	}

	var ev ghEvent
	if path := get("GITHUB_EVENT_PATH"); path != "" && env.ReadFile != nil {
		if data, err := env.ReadFile(path); err == nil {
			// A malformed payload is non-fatal: fall back to env-only fields.
			_ = json.Unmarshal(data, &ev)
		}
	}
	if ev.Repository.HTMLURL != "" {
		r.URL = ev.Repository.HTMLURL
	}
	if ev.Sender.Login != "" {
		r.Actor = ev.Sender.Login
	}

	switch get("GITHUB_EVENT_NAME") {
	case "pull_request", "pull_request_target":
		r.Verb = VerbPR
		r.Ref = ev.PullRequest.Head.Sha
		r.Repo = ev.PullRequest.Head.Repo.FullName
		r.URL = ev.PullRequest.Head.Repo.HTMLURL
		r.IsTag = false
	case "schedule":
		r.Verb = VerbSchedule
		r.Ref = firstNonEmpty(ev.Repository.DefaultBranch, "main")
		r.Depth = 0
		r.IsProduction = true
		r.IsTag = false
	case "workflow_dispatch":
		r.Verb = VerbDispatch
	default: // push and anything else behaves like a push
		r.Verb = VerbPush
	}
	return r, nil
}

// resolveGitLab maps GitLab's CI_* variables onto the same Resolution. It
// follows the GitHub mapping so a GitLab pipeline drives an identical checkout.
func resolveGitLab(env Env) Resolution {
	get := env.Getenv
	r := Resolution{
		CI:    PlatformGitLab,
		Depth: 1,
		Repo:  get("CI_PROJECT_PATH"),
		Ref:   firstNonEmpty(get("CI_COMMIT_REF_NAME"), get("CI_COMMIT_SHA")),
		URL:   get("CI_PROJECT_URL"),
		Actor: get("GITLAB_USER_LOGIN"),
		IsTag: get("CI_COMMIT_TAG") != "",
	}

	switch get("CI_PIPELINE_SOURCE") {
	case "merge_request_event":
		r.Verb = VerbPR
		r.Ref = firstNonEmpty(get("CI_MERGE_REQUEST_SOURCE_BRANCH_SHA"), get("CI_COMMIT_SHA"))
		r.Repo = firstNonEmpty(get("CI_MERGE_REQUEST_SOURCE_PROJECT_PATH"), get("CI_PROJECT_PATH"))
		r.URL = firstNonEmpty(get("CI_MERGE_REQUEST_SOURCE_PROJECT_URL"), get("CI_PROJECT_URL"))
		r.IsTag = false
	case "schedule":
		r.Verb = VerbSchedule
		r.Ref = firstNonEmpty(get("CI_DEFAULT_BRANCH"), "main")
		r.Depth = 0
		r.IsProduction = true
		r.IsTag = false
	case "web", "api", "trigger", "pipeline":
		r.Verb = VerbDispatch
	default: // push, etc.
		r.Verb = VerbPush
	}
	return r
}

// resolveLocal resolves from the developer's working copy so `clog ci resolve`
// shows exactly what CI would compute, before any push.
func resolveLocal(env Env) (Resolution, error) {
	git := env.Git
	if git == nil {
		git = execGit{}
	}
	r := Resolution{
		CI:    PlatformLocal,
		Verb:  VerbLocal,
		Depth: 1,
	}
	r.Ref = git.CurrentRef()
	r.Repo = git.RepoSlug()
	r.URL = git.RemoteURL()
	r.Actor = firstNonEmpty(git.UserName(), env.Getenv("USER"))
	r.IsTag = git.OnExactTag()
	return r, nil
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
