//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/

package ci

import (
	"os/exec"
	"regexp"
	"strings"
)

// GitResolver supplies the local-git facts the resolver needs in local mode.
// It is an interface so local-mode resolution can be tested with a fake.
type GitResolver interface {
	CurrentRef() string // branch name, or short SHA when detached
	RepoSlug() string   // owner/name parsed from the origin remote
	RemoteURL() string  // origin remote URL
	UserName() string   // git config user.name
	OnExactTag() bool   // HEAD points exactly at a tag
}

// execGit implements GitResolver by shelling out to git. All calls are
// best-effort: a missing repo or git binary yields empty strings, not errors,
// so `clog ci resolve` still prints a sensible LOCAL result outside a repo.
type execGit struct{}

func (execGit) git(args ...string) string {
	out, err := exec.Command("git", args...).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func (g execGit) CurrentRef() string {
	if ref := g.git("rev-parse", "--abbrev-ref", "HEAD"); ref != "" && ref != "HEAD" {
		return ref
	}
	return g.git("rev-parse", "--short", "HEAD")
}

func (g execGit) RemoteURL() string {
	return g.git("remote", "get-url", "origin")
}

func (g execGit) RepoSlug() string {
	return slugFromRemote(g.RemoteURL())
}

func (g execGit) UserName() string {
	return g.git("config", "user.name")
}

func (g execGit) OnExactTag() bool {
	return g.git("describe", "--exact-match", "--tags", "HEAD") != ""
}

// slugRe extracts owner/name from common git remote URL forms:
//
//	git@host:owner/name.git, https://host/owner/name.git, ssh://host/owner/name
var slugRe = regexp.MustCompile(`[:/]([^/:]+/[^/]+?)(?:\.git)?/?$`)

func slugFromRemote(url string) string {
	url = strings.TrimSpace(url)
	if url == "" {
		return ""
	}
	if m := slugRe.FindStringSubmatch(url); m != nil {
		return m[1]
	}
	return ""
}
