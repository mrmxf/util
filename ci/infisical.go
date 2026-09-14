//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/

package ci

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// httpClient is shared by the Infisical and OIDC calls; a var so tests can
// substitute one.
var httpClient = &http.Client{Timeout: 30 * time.Second}

// Secret is one fetched secret. Values never reach a log line.
type Secret struct {
	Key   string
	Value string
}

// infisicalLogin exchanges a platform OIDC token for a short-lived Infisical
// access token (POST /api/v1/auth/oidc-auth/login).
func infisicalLogin(domain, identityID, jwt string) (string, error) {
	body, _ := json.Marshal(map[string]string{"identityId": identityID, "jwt": jwt})
	req, err := http.NewRequest(http.MethodPost, strings.TrimRight(domain, "/")+"/api/v1/auth/oidc-auth/login", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	var out struct {
		AccessToken string `json:"accessToken"`
	}
	if err := doJSON(req, &out); err != nil {
		return "", fmt.Errorf("infisical OIDC login as identity %s: %w", identityID, err)
	}
	if out.AccessToken == "" {
		return "", fmt.Errorf("infisical OIDC login as identity %s: no access token in response", identityID)
	}
	return out.AccessToken, nil
}

// infisicalFetch reads the secrets of one env + folder, imports included
// (GET /api/v4/secrets), and merges them the way the Infisical CLI does: the
// folder's own secrets override imported ones, and a later import overrides
// an earlier one.
func infisicalFetch(domain, token, projectID, env, path string) ([]Secret, error) {
	q := url.Values{}
	q.Set("projectId", projectID)
	q.Set("environment", env)
	q.Set("secretPath", path)
	q.Set("includeImports", "true")
	q.Set("expandSecretReferences", "true")
	q.Set("recursive", "false")
	req, err := http.NewRequest(http.MethodGet, strings.TrimRight(domain, "/")+"/api/v4/secrets?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	type rawSecret struct {
		Key   string `json:"secretKey"`
		Value string `json:"secretValue"`
	}
	var out struct {
		Secrets []rawSecret `json:"secrets"`
		Imports []struct {
			Secrets []rawSecret `json:"secrets"`
		} `json:"imports"`
	}
	if err := doJSON(req, &out); err != nil {
		return nil, fmt.Errorf("infisical fetch env=%s path=%s: %w", env, path, err)
	}

	merged := []Secret{}
	seen := map[string]bool{}
	add := func(s rawSecret) {
		if s.Key == "" || seen[s.Key] {
			return
		}
		seen[s.Key] = true
		merged = append(merged, Secret{Key: s.Key, Value: s.Value})
	}
	for _, s := range out.Secrets {
		add(s)
	}
	for i := len(out.Imports) - 1; i >= 0; i-- {
		for _, s := range out.Imports[i].Secrets {
			add(s)
		}
	}
	return merged, nil
}

// doJSON performs req and decodes a JSON body into v. Error messages carry
// the HTTP status and Infisical's "message" field, never request credentials.
func doJSON(req *http.Request, v any) error {
	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%s %s: %w", req.Method, redactURL(req.URL), err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		var msg struct {
			Message string `json:"message"`
		}
		_ = json.Unmarshal(data, &msg)
		if msg.Message == "" {
			msg.Message = http.StatusText(resp.StatusCode)
		}
		return fmt.Errorf("%s %s: HTTP %d: %s", req.Method, redactURL(req.URL), resp.StatusCode, msg.Message)
	}
	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("%s %s: unexpected response: %w", req.Method, redactURL(req.URL), err)
	}
	return nil
}

// redactURL drops the query string, which may carry request tokens.
func redactURL(u *url.URL) string {
	c := *u
	c.RawQuery = ""
	return c.String()
}
