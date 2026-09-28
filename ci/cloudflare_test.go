//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/

package ci

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// fakeCloudflare answers the two Pages project calls. exists decides the GET;
// status overrides it (401/403). It records every call as "METHOD path".
type fakeCloudflare struct {
	mu      sync.Mutex
	exists  bool
	status  int
	calls   []string
	created map[string]string
}

func (f *fakeCloudflare) serve(t *testing.T) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		f.calls = append(f.calls, r.Method+" "+r.URL.Path)
		if r.Header.Get("Authorization") != "Bearer tok" {
			t.Errorf("no bearer token on %s", r.URL.Path)
		}
		switch {
		case f.status != 0:
			w.WriteHeader(f.status)
			_, _ = w.Write([]byte(`{"success":false,"errors":[{"code":10000,"message":"Authentication error"}]}`))
		case r.Method == http.MethodGet && f.exists:
			_, _ = w.Write([]byte(`{"success":true,"result":{"name":"site"}}`))
		case r.Method == http.MethodGet:
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"success":false,"errors":[{"code":8000007,"message":"Project not found"}]}`))
		case r.Method == http.MethodPost:
			_ = json.NewDecoder(r.Body).Decode(&f.created)
			_, _ = w.Write([]byte(`{"success":true,"result":{}}`))
		}
	}))
	prev := cfAPIBase
	cfAPIBase = srv.URL
	t.Cleanup(func() { cfAPIBase = prev; srv.Close() })
}

func pagesTarget(t *testing.T) (Config, Env) {
	t.Helper()
	cfg := Config{Targets: map[string]Target{"pages": {Kind: KindCloudflarePage, Dev: map[string]any{
		"dir": builtSite(t), "project": "site-staging", "production-branch": "main",
	}}}}
	env := laptop(map[string]string{"CLOUDFLARE_API_TOKEN": "tok", "CLOUDFLARE_ACCOUNT_ID": "acct"}, fakeGit{})
	return cfg, env
}

func TestPagesDeployUsesAnExistingProject(t *testing.T) {
	cf := &fakeCloudflare{exists: true}
	cf.serve(t)
	cfg, env := pagesTarget(t)
	calls, restore := recordExec(t, "")
	defer restore()

	if err := Deploy(env, cfg, ModeDev, "", false, &bytes.Buffer{}); err != nil {
		t.Fatalf("deploy: %v", err)
	}
	if strings.Join(cf.calls, ",") != "GET /accounts/acct/pages/projects/site-staging" {
		t.Errorf("cloudflare calls = %v, want one GET", cf.calls)
	}
	if !strings.Contains(strings.Join(*calls, "\n"), "wrangler pages deploy") {
		t.Errorf("no upload: %v", *calls)
	}
}

// The bonfire behaviour, now the default: a missing project is created, with
// its production branch, before the upload - and nothing touches a domain.
func TestPagesDeployCreatesAMissingProject(t *testing.T) {
	cf := &fakeCloudflare{}
	cf.serve(t)
	cfg, env := pagesTarget(t)
	calls, restore := recordExec(t, "")
	defer restore()

	var out bytes.Buffer
	if err := Deploy(env, cfg, ModeDev, "", false, &out); err != nil {
		t.Fatalf("deploy: %v", err)
	}
	if got := strings.Join(cf.calls, ","); got != "GET /accounts/acct/pages/projects/site-staging,POST /accounts/acct/pages/projects" {
		t.Errorf("cloudflare calls = %s", got)
	}
	if cf.created["name"] != "site-staging" || cf.created["production_branch"] != "main" {
		t.Errorf("created %v", cf.created)
	}
	for _, c := range cf.calls {
		if strings.Contains(c, "domains") || strings.Contains(c, "dns") {
			t.Errorf("a project create must not touch domains or DNS: %s", c)
		}
	}
	if !strings.Contains(out.String(), "created Pages project site-staging") {
		t.Errorf("the create should be said out loud: %q", out.String())
	}
	if !strings.Contains(strings.Join(*calls, "\n"), "--project-name=site-staging") {
		t.Errorf("no upload after the create: %v", *calls)
	}
}

// A token that cannot read Pages stops before wrangler runs, and says why.
func TestPagesDeployStopsOnARefusedToken(t *testing.T) {
	cf := &fakeCloudflare{status: http.StatusForbidden}
	cf.serve(t)
	cfg, env := pagesTarget(t)
	calls, restore := recordExec(t, "")
	defer restore()

	err := Deploy(env, cfg, ModeDev, "", false, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "Cloudflare Pages > Edit") {
		t.Errorf("err = %v, want the permission named", err)
	}
	if len(*calls) != 0 {
		t.Errorf("wrangler ran anyway: %v", *calls)
	}
}

func TestPagesDryRunCreatesNothing(t *testing.T) {
	cf := &fakeCloudflare{}
	cf.serve(t)
	cfg, env := pagesTarget(t)
	calls, restore := recordExec(t, "")
	defer restore()

	var out bytes.Buffer
	if err := Deploy(env, cfg, ModeDev, "", true, &out); err != nil {
		t.Fatalf("dry run: %v", err)
	}
	for _, c := range cf.calls {
		if strings.HasPrefix(c, "POST") {
			t.Errorf("dry run created a project: %s", c)
		}
	}
	if strings.Contains(strings.Join(*calls, "\n"), "wrangler") {
		t.Errorf("dry run uploaded: %v", *calls)
	}
	if !strings.Contains(out.String(), "would create Pages project site-staging") {
		t.Errorf("dry run should say it would create: %q", out.String())
	}
}
