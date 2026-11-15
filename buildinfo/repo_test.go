//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/

package buildinfo

import (
	"strings"
	"testing"
)

func TestParseGitHubRemote(t *testing.T) {
	cases := map[string]string{
		"https://github.com/mrmxf/clog.git":     "mrmxf/clog",
		"https://github.com/mrmxf/clog":         "mrmxf/clog",
		"git@github.com:mrmxf/clog-mrmxf.git":   "mrmxf/clog-mrmxf",
		"ssh://git@github.com/someone/clog.git": "someone/clog",
		"https://gitlab.com/mrmxf/clog.git":     "",
		"https://github.example.com/mrmxf/clog": "",
		"https://notgithub.com/mrmxf/clog":      "",
		"":                                      "",
	}
	for in, want := range cases {
		if got := parseGitHubRemote(in); got != want {
			t.Errorf("parseGitHubRemote(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestReleaseRepoPrefersGitHubActions(t *testing.T) {
	t.Setenv("GITHUB_REPOSITORY", "someone/clog")
	t.Setenv("GITHUB_SERVER_URL", "https://github.com")
	if got := ReleaseRepo("."); got != "someone/clog" {
		t.Errorf("ReleaseRepo = %q, want someone/clog", got)
	}
	t.Setenv("GITHUB_SERVER_URL", "https://ghes.example.com")
	if got := ReleaseRepo("."); got != "" {
		t.Errorf("ReleaseRepo on GHES = %q, want \"\"", got)
	}
}

func TestRepoRoundTripsThroughLinkerData(t *testing.T) {
	d := LinkerDataJSON{Build: "prod", Tag: "v1.2.3", Repo: "mrmxf/clog"}
	info, _, err := ParseLinkerJSON(d.JSON())
	if err != nil {
		t.Fatal(err)
	}
	if info.Repo != "mrmxf/clog" {
		t.Errorf("Repo = %q, want mrmxf/clog", info.Repo)
	}
	if j := (LinkerDataJSON{Tag: "v1"}).JSON(); strings.Contains(j, "repo") {
		t.Errorf("an empty Repo must be omitted, got %s", j)
	}
}
