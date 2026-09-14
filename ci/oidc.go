//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/

package ci

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// GitLabIDTokenVar is the id_tokens variable a GitLab job must declare:
//
//	id_tokens:
//	  INFISICAL_ID_TOKEN: {aud: https://eu.infisical.com}
const GitLabIDTokenVar = "INFISICAL_ID_TOKEN"

// platformJWT returns the OIDC token the CI platform minted for this job.
func platformJWT(env Env, platform Platform, audience string) (string, error) {
	switch platform {
	case PlatformGitHub:
		return githubJWT(env, audience)
	case PlatformGitLab:
		jwt := strings.TrimSpace(env.Getenv(GitLabIDTokenVar))
		if jwt == "" {
			return "", fmt.Errorf("$%s is empty: add to the job  id_tokens: {%s: {aud: %s}}", GitLabIDTokenVar, GitLabIDTokenVar, audience)
		}
		return jwt, nil
	default:
		return "", fmt.Errorf("no OIDC token source on platform %q", platform)
	}
}

// githubJWT requests an ID token from the Actions runtime with the given
// audience. Both request variables exist only when the job (and, for a reusable
// workflow, its caller) grants `permissions: id-token: write`.
func githubJWT(env Env, audience string) (string, error) {
	reqURL := env.Getenv("ACTIONS_ID_TOKEN_REQUEST_URL")
	reqToken := env.Getenv("ACTIONS_ID_TOKEN_REQUEST_TOKEN")
	if reqURL == "" || reqToken == "" {
		return "", fmt.Errorf("GitHub OIDC unavailable: add  permissions: {id-token: write}  to the workflow and the reusable job")
	}
	u, err := url.Parse(reqURL)
	if err != nil {
		return "", fmt.Errorf("ACTIONS_ID_TOKEN_REQUEST_URL is not a URL: %w", err)
	}
	q := u.Query()
	q.Set("audience", audience)
	u.RawQuery = q.Encode()

	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "bearer "+reqToken)

	var out struct {
		Value string `json:"value"`
	}
	if err := doJSON(req, &out); err != nil {
		return "", fmt.Errorf("GitHub OIDC token request: %w", err)
	}
	if out.Value == "" {
		return "", fmt.Errorf("GitHub OIDC token request returned no token")
	}
	return out.Value, nil
}
