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
)

// cfAPIBase is the Cloudflare v4 API, replaceable in tests.
var cfAPIBase = "https://api.cloudflare.com/client/v4"

// cfResponse is the envelope every Cloudflare v4 call answers with.
type cfResponse struct {
	Success bool `json:"success"`
	Errors  []struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"errors"`
	Result json.RawMessage `json:"result"`
}

// cfErrors renders the envelope's errors for a message.
func (r cfResponse) cfErrors() string {
	var b bytes.Buffer
	for i, e := range r.Errors {
		if i > 0 {
			b.WriteString("; ")
		}
		fmt.Fprintf(&b, "%d: %s", e.Code, e.Message)
	}
	return b.String()
}

// cfCall makes one Cloudflare API call and returns the HTTP status and the
// decoded envelope. A body that is not the envelope is an error.
func cfCall(token, method, path string, body any) (int, cfResponse, error) {
	var rd io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return 0, cfResponse{}, err
		}
		rd = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, cfAPIBase+path, rd)
	if err != nil {
		return 0, cfResponse{}, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return 0, cfResponse{}, fmt.Errorf("cloudflare %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()
	var r cfResponse
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return resp.StatusCode, cfResponse{}, fmt.Errorf("cloudflare %s %s: HTTP %d, unreadable reply: %w", method, path, resp.StatusCode, err)
	}
	return resp.StatusCode, r, nil
}

// EnsurePagesProject makes sure a Cloudflare Pages project exists, creating it
// with its production branch when it does not. It reports whether it created
// one. With dryRun it only says what it would do.
//
// It never attaches a domain or writes DNS. A new project serves only its own
// *.pages.dev host, so creating one cannot take a live site anywhere - which
// is why this is safe to do by default. Moving a domain is a separate,
// deliberate act (a cutover).
func EnsurePagesProject(token, account, project, prodBranch string, dryRun bool, out io.Writer) (bool, error) {
	path := "/accounts/" + url.PathEscape(account) + "/pages/projects/" + url.PathEscape(project)
	status, r, err := cfCall(token, http.MethodGet, path, nil)
	if err != nil {
		return false, err
	}
	switch {
	case status == http.StatusOK && r.Success:
		return false, nil
	case status != http.StatusNotFound:
		// 401/403 is a token without Pages:Read/Edit; say so rather than
		// trying to create and failing less clearly.
		return false, fmt.Errorf("cannot read Pages project %s (HTTP %d: %s) - the token needs Account > Cloudflare Pages > Edit",
			project, status, r.cfErrors())
	}

	if dryRun {
		fmt.Fprintf(out, "dry-run: would create Pages project %s (production branch %s)\n", project, prodBranch)
		return true, nil
	}
	status, r, err = cfCall(token, http.MethodPost, "/accounts/"+url.PathEscape(account)+"/pages/projects",
		map[string]string{"name": project, "production_branch": prodBranch})
	if err != nil {
		return false, err
	}
	if !r.Success {
		return false, fmt.Errorf("cannot create Pages project %s (HTTP %d: %s)", project, status, r.cfErrors())
	}
	fmt.Fprintf(out, "created Pages project %s (production branch %s)\n", project, prodBranch)
	return true, nil
}
