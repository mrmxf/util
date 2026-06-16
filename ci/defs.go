//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/

// Package ci normalizes the environment of whichever CI system is active
// (GitHub Actions, GitLab CI, or a local developer machine) into one explicit
// struct. Build code reads that struct instead of raw GITHUB_*/CI_* variables,
// so build behavior becomes environment-shape-independent — the core lever for
// minimizing dev↔CI drift (see Phase 9½, Workstream A).
package ci

// Platform identifies which CI system (if any) the resolver detected.
type Platform string

const (
	PlatformGitHub Platform = "github"
	PlatformGitLab Platform = "gitlab"
	PlatformLocal  Platform = "local"
)

// Verb is the normalized trigger class. It is intentionally CI-agnostic so the
// same downstream logic drives a checkout/build on either platform.
type Verb string

const (
	// VerbPR — a pull/merge request. Checks out the head of the requesting
	// fork at a specific SHA (shallow).
	VerbPR Verb = "PR"
	// VerbPush — a branch push. Checks out the pushed ref (shallow).
	VerbPush Verb = "PUSH"
	// VerbSchedule — a scheduled run. Resolves to a production checkout of the
	// default branch with full history (depth 0).
	VerbSchedule Verb = "SCHEDULE"
	// VerbDispatch — a manual trigger (workflow_dispatch / web / api / trigger).
	// Resolves refs identically to PUSH; kept distinct for observability.
	VerbDispatch Verb = "DISPATCH"
	// VerbLocal — neither CI system is active; resolved from local git.
	VerbLocal Verb = "LOCAL"
)

// Resolution is the normalized, CI-agnostic description of the current event.
// It is the single source of truth a build step reads to decide what to check
// out and how to behave.
type Resolution struct {
	CI    Platform `json:"ci"`
	Verb  Verb     `json:"verb"`
	Ref   string   `json:"ref"`   // git ref or SHA to check out
	Repo  string   `json:"repo"`  // owner/name to check out from
	URL   string   `json:"url"`   // human URL of the repo, for logging
	Depth int      `json:"depth"` // checkout fetch-depth (0 = full history, 1 = shallow)

	Actor        string `json:"actor"`         // who triggered the event
	IsProduction bool   `json:"is_production"` // true → check out the production tag
	IsTag        bool   `json:"is_tag"`        // true → the ref is a tag
}
