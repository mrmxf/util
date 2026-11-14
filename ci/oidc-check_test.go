//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/

package ci

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// fakeJWT builds an unsigned token with the given claims.
func fakeJWT(t *testing.T, claims map[string]any) string {
	t.Helper()
	data, err := json.Marshal(claims)
	if err != nil {
		t.Fatal(err)
	}
	return "eyJhbGciOiJSUzI1NiJ9." + base64.RawURLEncoding.EncodeToString(data) + ".c2lnbmF0dXJl"
}

func TestLoginErrorNamesClaimsNotToken(t *testing.T) {
	jwt := fakeJWT(t, map[string]any{
		"sub": "repo:mrmxf@19326048/nabu@1444420835:ref:refs/heads/main",
		"aud": "https://eu.infisical.com", "ref": "refs/heads/main",
	})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"message":"Access denied: OIDC subject not allowed."}`))
	}))
	defer srv.Close()

	_, err := infisicalLogin(srv.URL, testDevUUID, jwt)
	if err == nil {
		t.Fatal("want an error")
	}
	msg := err.Error()
	for _, want := range []string{
		"OIDC subject not allowed",
		`sub="repo:mrmxf@19326048/nabu@1444420835:ref:refs/heads/main"`,
		`aud="https://eu.infisical.com"`,
		"clog CI oidc",
	} {
		if !strings.Contains(msg, want) {
			t.Errorf("error lacks %q:\n%s", want, msg)
		}
	}
	if strings.Contains(msg, jwt) || strings.Contains(msg, strings.Split(jwt, ".")[1]) {
		t.Error("error leaks the JWT")
	}
}

func TestExpectedSubjects(t *testing.T) {
	dev, prod := expectedSubjects(PlatformGitHub, subjectPrefix("repo:mrmxf@19326048/nabu@1444420835:ref:refs/heads/main"))
	if dev != "repo:mrmxf@19326048/nabu@1444420835:ref:refs/*" || prod != "repo:mrmxf@19326048/nabu@1444420835:ref:refs/tags/v*" {
		t.Errorf("github: %s / %s", dev, prod)
	}
	dev, prod = expectedSubjects(PlatformGitLab, subjectPrefix("project_path:mrmxf/site:ref_type:branch:ref:main"))
	if dev != "project_path:mrmxf/site:ref_type:*:ref:*" || prod != "project_path:mrmxf/site:ref_type:tag:ref:v*" {
		t.Errorf("gitlab: %s / %s", dev, prod)
	}
}

func TestGlobMatch(t *testing.T) {
	for _, tc := range []struct {
		pattern, subject string
		want             bool
	}{
		{"repo:o/r:ref:refs/*", "repo:o/r:ref:refs/heads/feature/x", true},
		{"repo:o/r:ref:refs/tags/v*", "repo:o/r:ref:refs/heads/main", false},
		{"project_path:g/{a,b}:ref_type:*:ref:*", "project_path:g/b:ref_type:tag:ref:v1", true},
		{"project_path:g/{a}:ref_type:*:ref:*", "project_path:g/a:ref_type:tag:ref:v1", false}, // one-item brace is literal
		{"project_path:g/p:ref_type:{tag:ref:v*,branch:ref:main}", "project_path:g/p:ref_type:branch:ref:main", true},
		{"project_path:g/p:ref_type:{tag:ref:v*,branch:ref:main}", "project_path:g/p:ref_type:branch:ref:dev", false},
	} {
		if got := globMatch(tc.pattern, tc.subject); got != tc.want {
			t.Errorf("globMatch(%q, %q) = %v, want %v", tc.pattern, tc.subject, got, tc.want)
		}
	}
}

func TestLintSubject(t *testing.T) {
	const imm = "repo:mrmxf@19326048/nabu@1444420835"
	for _, tc := range []struct {
		name, pattern string
		platform      Platform
		mode          string
		want          string // substring of one problem; "" = no problems
	}{
		{"good dev", imm + ":ref:refs/*", PlatformGitHub, ModeDev, ""},
		{"good prod", imm + ":ref:refs/tags/v*", PlatformGitHub, ModeProd, ""},
		{"environment prod", "repo:mrmxf/nabu:environment:prod", PlatformGitHub, ModeProd, ""},
		{"leading space (2026-11-14)", " " + imm + ":ref:refs/*", PlatformGitHub, ModeDev, "whitespace"},
		{"one-item brace", "repo:mrmxf/{nabu}:ref:refs/*", PlatformGitHub, ModeDev, "matched literally"},
		{"top-level comma", "repo:mrmxf/nabu:ref:refs/*,repo:mrmxf/x:ref:refs/*", PlatformGitHub, ModeDev, "comma outside braces"},
		{"owner-wide", "repo:mrmxf/*", PlatformGitHub, ModeDev, "admits other repos"},
		{"other repo", "repo:mrmxf/clog:ref:refs/*", PlatformGitHub, ModeDev, "does not name this repo"},
		{"prod admits branches", imm + ":ref:refs/*", PlatformGitHub, ModeProd, "admits any branch"},
		{"prod ref star (2026-09 bonfire)", "project_path:mrmxf/site:ref_type:*:ref:*", PlatformGitLab, ModeProd, "admits any branch"},
		{"gitlab prod tag+main", "project_path:mrmxf/{site,nabu}:ref_type:{tag:ref:v*,branch:ref:main}", PlatformGitLab, ModeProd, ""},
		{"pasted twice (2026-09 bonfire)", "project_path:mrmxf/nabu:ref_type:*:ref:ref_type:branch:ref:main", PlatformGitLab, ModeDev, ""},
		{"empty", "", PlatformGitHub, ModeDev, "no Subject"},
		{"repo name is a prefix of another", "repo:mrmxf/nabu-tutorial:ref:refs/*", PlatformGitHub, ModeDev, "does not name this repo"},
	} {
		got := lintSubject(tc.pattern, tc.platform, tc.mode, "mrmxf/nabu")
		joined := strings.Join(got, " | ")
		if tc.want == "" && len(got) > 0 {
			t.Errorf("%s: want no problems, got %s", tc.name, joined)
		}
		if tc.want != "" && !strings.Contains(joined, tc.want) {
			t.Errorf("%s: want a problem containing %q, got %q", tc.name, tc.want, joined)
		}
	}
}

func TestOIDCOnLaptopReadsBoundSubjects(t *testing.T) {
	bound := map[string]string{
		testDevUUID:  " repo:mrmxf/nabu:ref:refs/*", // the leading-space bug
		testProdUUID: "repo:mrmxf/nabu:ref:refs/tags/v*",
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer user-token" {
			http.Error(w, `{"message":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		uuid := strings.TrimPrefix(r.URL.Path, "/api/v1/auth/oidc-auth/identities/")
		_ = json.NewEncoder(w).Encode(map[string]any{"identityOidcAuth": map[string]string{
			"boundSubject": bound[uuid], "boundAudiences": "https://eu.infisical.com",
		}})
	}))
	defer srv.Close()
	oldTok := userToken
	userToken = func(string) (string, error) { return "user-token", nil }
	t.Cleanup(func() { userToken = oldTok })

	inf := testConfig(srv.URL).Infisical
	delete(inf.Identity, "gitlab") // a GitHub-only repo
	var out bytes.Buffer
	err := oidcOnLaptop(Resolution{Repo: "mrmxf/nabu"}, inf, &out)
	if err == nil || !strings.Contains(err.Error(), "1 identity check(s) failed") {
		t.Fatalf("want one failed check, got %v\n%s", err, out.String())
	}
	if !strings.Contains(out.String(), "FAIL github dev-uuid") || !strings.Contains(out.String(), "ok   github prod-uuid") {
		t.Errorf("unexpected report:\n%s", out.String())
	}
	if strings.Contains(out.String(), "user-token") {
		t.Error("report leaks the user token")
	}
}
