//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/

package install

import (
	"errors"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

// githubAPI is the GitHub REST API base URL; a var so tests can point it at a
// local server.
var githubAPI = "https://api.github.com"

// githubTokenEnvs are checked in order for a token to authenticate API calls,
// which raises the rate limit from 60 to 5000 requests an hour.
var githubTokenEnvs = []string{"GITHUB_TOKEN", "GH_TOKEN", "GHAT"}

// rateLimitOnce makes the rate-limit warning appear once per run, however many
// API calls hit the limit.
var rateLimitOnce sync.Once

// githubToken returns the first token found and the env var it came from.
func githubToken() (token, env string) {
	for _, env := range githubTokenEnvs {
		if t := os.Getenv(env); t != "" {
			return t, env
		}
	}
	return "", ""
}

// githubGet performs a GitHub API GET, authenticated when a token is set. When
// the rate limit is exhausted it logs a warning plus how to fix it, closes the
// body and returns an error.
func githubGet(url string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	token, env := githubToken()
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
		vlog("  github api authenticated", "token-env", env)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	if isRateLimited(resp) {
		resp.Body.Close()
		reportRateLimit(resp, env)
		return nil, errors.New("GitHub API rate limit reached")
	}
	return resp, nil
}

// isRateLimited reports whether resp is GitHub's primary rate-limit refusal:
// 403 or 429 with no requests remaining.
func isRateLimited(resp *http.Response) bool {
	if resp.StatusCode != http.StatusForbidden && resp.StatusCode != http.StatusTooManyRequests {
		return false
	}
	return resp.Header.Get("X-RateLimit-Remaining") == "0"
}

// reportRateLimit logs the rate-limit warning and the token fix, once per run.
func reportRateLimit(resp *http.Response, tokenEnv string) {
	rateLimitOnce.Do(func() {
		attrs := []any{"limit", resp.Header.Get("X-RateLimit-Limit")}
		if reset, err := strconv.ParseInt(resp.Header.Get("X-RateLimit-Reset"), 10, 64); err == nil {
			at := time.Unix(reset, 0)
			attrs = append(attrs, "resets", at.Format("15:04:05"), "in", time.Until(at).Round(time.Second).String())
		}

		if tokenEnv == "" {
			slog.Warn("GitHub API rate limit reached for unauthenticated requests", attrs...)
			slog.Info("fix: set a GitHub token to raise the limit to 5000/hour, e.g. export GITHUB_TOKEN=\"$(gh auth token)\"",
				"ci", "env: GITHUB_TOKEN: ${{ github.token }}", "also-read", "GH_TOKEN, GHAT")
			return
		}
		slog.Warn("GitHub API rate limit reached", append(attrs, "token-env", tokenEnv)...)
		slog.Info("fix: the token in " + tokenEnv + " is already in use; wait for the reset or use a token with a higher limit")
	})
}
