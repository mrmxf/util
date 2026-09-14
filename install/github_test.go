//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/

package install

import (
	"bytes"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// fakeGitHub serves a rate-limit refusal for every request and records the
// Authorization header it received.
func fakeGitHub(t *testing.T) (gotAuth *string) {
	t.Helper()
	gotAuth = new(string)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*gotAuth = r.Header.Get("Authorization")
		w.Header().Set("X-RateLimit-Limit", "60")
		w.Header().Set("X-RateLimit-Remaining", "0")
		w.Header().Set("X-RateLimit-Reset", fmt.Sprint(time.Now().Add(10*time.Minute).Unix()))
		w.WriteHeader(http.StatusForbidden)
	}))
	t.Cleanup(srv.Close)
	prev := githubAPI
	githubAPI = srv.URL
	t.Cleanup(func() { githubAPI = prev })
	rateLimitOnce = sync.Once{}
	return gotAuth
}

func captureLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})))
	t.Cleanup(func() { slog.SetDefault(prev) })
	return &buf
}

func clearTokens(t *testing.T) {
	for _, env := range githubTokenEnvs {
		t.Setenv(env, "")
	}
}

func TestRateLimitUnauthenticated(t *testing.T) {
	clearTokens(t)
	gotAuth := fakeGitHub(t)
	logs := captureLog(t)

	spec := &VersionSpec{Strategy: "github-latest", Repo: "o/r", TagPrefix: "v"}
	if _, err := ResolveVersion(spec); err == nil || !strings.Contains(err.Error(), "rate limit") {
		t.Fatalf("expected rate limit error, got %v", err)
	}
	// a second limited call in the same run must not repeat the warning
	if _, err := resolveGitHubTags(&VersionSpec{Strategy: "github-tags", Repo: "o/r"}); err == nil {
		t.Fatal("expected rate limit error from tags")
	}

	out := logs.String()
	if *gotAuth != "" {
		t.Errorf("sent Authorization %q without a token", *gotAuth)
	}
	if n := strings.Count(out, "level=WARN msg=\"GitHub API rate limit reached"); n != 1 {
		t.Errorf("want exactly one rate limit warning, got %d:\n%s", n, out)
	}
	if !strings.Contains(out, "level=INFO msg=\"fix: set a GitHub token") || !strings.Contains(out, "GITHUB_TOKEN") {
		t.Errorf("missing token fix INF line:\n%s", out)
	}
}

func TestRateLimitWithToken(t *testing.T) {
	clearTokens(t)
	t.Setenv("GH_TOKEN", "secret")
	gotAuth := fakeGitHub(t)
	logs := captureLog(t)

	if _, _, err := resolveGitHubReleaseAsset(&DownloadSpec{Repo: "o/r"}, unknownVersion); err == nil {
		t.Fatal("expected rate limit error")
	}
	if *gotAuth != "Bearer secret" {
		t.Errorf("Authorization = %q, want Bearer token from GH_TOKEN", *gotAuth)
	}
	out := logs.String()
	if !strings.Contains(out, "token-env=GH_TOKEN") || !strings.Contains(out, "already in use") {
		t.Errorf("expected authenticated rate limit messages:\n%s", out)
	}
	if strings.Contains(out, "secret") {
		t.Error("token value leaked into logs")
	}
}

func TestIsRateLimited(t *testing.T) {
	mk := func(code int, remaining string) *http.Response {
		r := &http.Response{StatusCode: code, Header: http.Header{}}
		if remaining != "" {
			r.Header.Set("X-RateLimit-Remaining", remaining)
		}
		return r
	}
	cases := []struct {
		resp *http.Response
		want bool
	}{
		{mk(403, "0"), true},
		{mk(429, "0"), true},
		{mk(403, "12"), false}, // forbidden for another reason
		{mk(404, "0"), false},
		{mk(200, "0"), false},
	}
	for _, c := range cases {
		if got := isRateLimited(c.resp); got != c.want {
			t.Errorf("status %d remaining %q: got %v", c.resp.StatusCode, c.resp.Header.Get("X-RateLimit-Remaining"), got)
		}
	}
}
