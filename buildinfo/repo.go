//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

package buildinfo

import (
	"os"
	"regexp"
	"strings"
)

// githubRemoteRe pulls owner/repo out of the remote URL forms git accepts for
// github.com: https://github.com/o/r(.git), git@github.com:o/r(.git) and
// ssh://git@github.com/o/r(.git).
var githubRemoteRe = regexp.MustCompile(`(?:^|[@/])github\.com[:/]+([^/\s]+)/([^/\s]+?)(?:\.git)?/?$`)

// ReleaseRepo is the GitHub owner/repo a build in dir is released from, for
// stamping into LinkerDataJSON.Repo. It is "" when there is no GitHub answer.
//
//  1. GITHUB_REPOSITORY, when running on github.com Actions. That is the repo
//     whose release job publishes the binaries.
//  2. otherwise the owner/repo of dir's origin remote, when it is on github.com:
//     a laptop build of a fork names the fork.
//
// A GitLab or GHES build gets "" rather than a guess, and its binary then
// falls back to the installer's default repo.
func ReleaseRepo(dir string) string {
	if r := os.Getenv("GITHUB_REPOSITORY"); r != "" {
		if s := os.Getenv("GITHUB_SERVER_URL"); s == "" || s == "https://github.com" {
			return r
		}
		return ""
	}
	url, err := git(dir, "remote", "get-url", "origin")
	if err != nil {
		return ""
	}
	return parseGitHubRemote(url)
}

// parseGitHubRemote returns owner/repo for a github.com remote URL, else "".
func parseGitHubRemote(url string) string {
	m := githubRemoteRe.FindStringSubmatch(strings.TrimSpace(url))
	if m == nil {
		return ""
	}
	return m[1] + "/" + m[2]
}
