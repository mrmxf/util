//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/

package ci

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// No test waits out the real poll.
func init() { probeSleep = func(time.Duration) {} }

func withVersion(t *testing.T, v string) {
	t.Helper()
	prev := ReleaseVersion
	ReleaseVersion = func() string { return v }
	t.Cleanup(func() { ReleaseVersion = prev })
}

func withPages(t *testing.T, out string, err error) {
	t.Helper()
	prev := pagesSettings
	pagesSettings = func(string) (string, error) { return out, err }
	t.Cleanup(func() { pagesSettings = prev })
}

// pihuw's incident: Pages built from a workflow, so pushes to gh-pages changed
// nothing. The deploy must refuse before it pushes, and say how to fix it.
func TestDeployGitHubPagesRefusesWhenPagesServesSomethingElse(t *testing.T) {
	for name, settings := range map[string]string{
		"workflow build": `{"build_type":"workflow","source":{"branch":"main","path":"/"}}`,
		"other branch":   `{"build_type":"legacy","source":{"branch":"main","path":"/"}}`,
		"docs folder":    `{"build_type":"legacy","source":{"branch":"gh-pages","path":"/docs"}}`,
	} {
		t.Run(name, func(t *testing.T) {
			withPages(t, settings, nil)
			calls, restore := recordExec(t, "")
			defer restore()
			err := Deploy(pagesEnv(nil), pagesConfig(map[string]any{"dir": builtSite(t)}), ModeProd, "", false, &bytes.Buffer{})
			if err == nil || !strings.Contains(err.Error(), "change nothing live") ||
				!strings.Contains(err.Error(), "Deploy from a branch → gh-pages") {
				t.Fatalf("want a refusal naming the fix, got %v", err)
			}
			if strings.Contains(strings.Join(*calls, "\n"), "push") {
				t.Errorf("nothing may be pushed, got %v", *calls)
			}
		})
	}
}

func TestDeployGitHubPagesPublishesWhenPagesServesTheBranch(t *testing.T) {
	withPages(t, `{"build_type":"legacy","source":{"branch":"gh-pages","path":"/"}}`, nil)
	withVersion(t, "v1.2.3")
	calls, restore := recordExec(t, "")
	defer restore()
	dir := builtSite(t)
	var out bytes.Buffer
	if err := Deploy(pagesEnv(nil), pagesConfig(map[string]any{"dir": dir}), ModeProd, "", false, &out); err != nil {
		t.Fatalf("deploy: %v", err)
	}
	if !strings.Contains(strings.Join(*calls, "\n"), "push --force") {
		t.Errorf("want a push, got %v", *calls)
	}
	if strings.Contains(out.String(), "not enabled") {
		t.Errorf("Pages is enabled; no enable-it message wanted: %q", out.String())
	}
	b, err := os.ReadFile(filepath.Join(dir, releaseMarker))
	if err != nil || strings.TrimSpace(string(b)) != "v1.2.3" {
		t.Errorf("%s = %q, %v; want v1.2.3", releaseMarker, b, err)
	}
}

// Pages off (or unreadable by this token) is a first deploy, not an error: the
// branch has to exist before Pages can be pointed at it.
func TestDeployGitHubPagesNotEnabledPublishesAndSaysSo(t *testing.T) {
	withPages(t, "gh: Not Found (HTTP 404)", errors.New("exit 1"))
	_, restore := recordExec(t, "")
	defer restore()
	var out bytes.Buffer
	if err := Deploy(pagesEnv(nil), pagesConfig(map[string]any{"dir": builtSite(t)}), ModeProd, "", false, &out); err != nil {
		t.Fatalf("deploy: %v", err)
	}
	if !strings.Contains(out.String(), "Deploy from a branch → gh-pages") {
		t.Errorf("want the enable-it message, got %q", out.String())
	}
}

func TestDeployGitHubPagesUnreadableSettingsStillPublishes(t *testing.T) {
	withPages(t, "gh: Resource not accessible by integration (HTTP 403)", errors.New("exit 1"))
	calls, restore := recordExec(t, "")
	defer restore()
	if err := Deploy(pagesEnv(nil), pagesConfig(map[string]any{"dir": builtSite(t)}), ModeProd, "", false, &bytes.Buffer{}); err != nil {
		t.Fatalf("an unreadable setting must not block the deploy: %v", err)
	}
	if !strings.Contains(strings.Join(*calls, "\n"), "push --force") {
		t.Errorf("want a push, got %v", *calls)
	}
}

func markerURL(v string) string {
	return "https://example.org/" + releaseMarker + "?clog=" + strings.ReplaceAll(v, "+", "%2B")
}

func TestProbeServedBuildMatches(t *testing.T) {
	withVersion(t, "v1.2.3+dev.1.gabc")
	stubHTTP(t, map[string]struct {
		status int
		body   string
	}{
		"https://example.org":          route(200, "<html>"),
		markerURL("v1.2.3+dev.1.gabc"): route(200, "v1.2.3+dev.1.gabc\n"),
	})
	got := probeWebsite(probeDeployment(KindGitHubPages, map[string]any{"domain": "example.org"}))
	if l := levelOf(got, "serving this build"); l != ProbePass {
		t.Errorf("serving this build = %q, want pass: %+v", l, got)
	}
}

// The check a 200 could never make: the site is up, and it is the wrong build.
func TestProbeServedBuildCatchesStaleSite(t *testing.T) {
	withVersion(t, "v1.2.4")
	stubHTTP(t, map[string]struct {
		status int
		body   string
	}{
		"https://example.org": route(200, "<html>"),
		markerURL("v1.2.4"):   route(200, "v1.2.3\n"),
	})
	got := probeWebsite(probeDeployment(KindGitHubPages, map[string]any{"domain": "example.org"}))
	if l := levelOf(got, "serving this build"); l != ProbeFail {
		t.Errorf("stale site = %q, want fail", l)
	}
}

func TestProbeServedBuildWithoutMarkerWarns(t *testing.T) {
	withVersion(t, "v1.2.3")
	stubHTTP(t, map[string]struct {
		status int
		body   string
	}{"https://example.org": route(200, "<html>")})
	got := probeWebsite(probeDeployment(KindGitHubPages, map[string]any{"domain": "example.org"}))
	if l := levelOf(got, "serving this build"); l != ProbeWarn {
		t.Errorf("no marker = %q, want warn", l)
	}
}

// A CDN can lag the push; a version that arrives mid-poll is a pass.
func TestProbeServedBuildWaitsForPropagation(t *testing.T) {
	withVersion(t, "v1.2.4")
	n := 0
	orig := httpGet
	httpGet = func(u string) (int, string, error) {
		if strings.Contains(u, releaseMarker) {
			n++
			if n < 3 {
				return 200, "v1.2.3", nil
			}
			return 200, "v1.2.4", nil
		}
		return 200, "<html>", nil
	}
	t.Cleanup(func() { httpGet = orig })
	f := probeServedBuild(probeDeployment(KindGitHubPages, nil), "https://example.org")
	if f.Level != ProbePass || n != 3 {
		t.Errorf("got %q after %d polls, want pass after 3", f.Level, n)
	}
}
