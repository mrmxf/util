//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/

package ci

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

// An Infisical OIDC 403 says the same thing for every cause ("OIDC subject not
// allowed"), and the console shows what someone meant to bind rather than what
// matches. This file turns that into something a person can act on:
//
//   - a failed login names the claims the token presented (never the token)
//   - `clog CI oidc` prints those claims and the Subject patterns to bind in a
//     CI job, and on a laptop reads each bound Subject back and checks it

// claimKeys are the claims worth showing: what an identity's Subject and
// claims are matched against. Order is the print order.
var claimKeys = []string{"sub", "aud", "iss", "ref", "ref_type", "ref_protected", "environment", "job_workflow_ref"}

// tokenClaims decodes a JWT's payload. It does not verify the signature: the
// claims are only shown to a person, never trusted.
func tokenClaims(jwt string) (map[string]any, error) {
	parts := strings.Split(jwt, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("not a JWT")
	}
	data, err := base64.RawURLEncoding.DecodeString(strings.TrimRight(parts[1], "="))
	if err != nil {
		return nil, fmt.Errorf("JWT payload: %w", err)
	}
	var c map[string]any
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("JWT payload: %w", err)
	}
	return c, nil
}

// claimSummary is `sub="…" aud="…" …` for the claimKeys present in c.
func claimSummary(c map[string]any) string {
	var out []string
	for _, k := range claimKeys {
		if v, ok := c[k]; ok {
			out = append(out, fmt.Sprintf("%s=%q", k, fmt.Sprint(v)))
		}
	}
	return strings.Join(out, " ")
}

// loginHint is appended to a failed OIDC login: what the token presented, and
// where to look next. The JWT itself is a live credential and never appears.
func loginHint(jwt string) string {
	c, err := tokenClaims(jwt)
	if err != nil {
		return "(token claims unreadable: " + err.Error() + ")"
	}
	return "(the token presented " + claimSummary(c) +
		"; the identity's OIDC Subject must match sub exactly or as a glob - check what is bound with: clog CI oidc)"
}

// subjectPrefix is the repo part of a subject: `repo:<owner>/<repo>` on GitHub
// (or `repo:<owner>@<id>/<repo>@<id>` for an immutable subject),
// `project_path:<group>/<project>` on GitLab.
func subjectPrefix(sub string) string {
	parts := strings.SplitN(sub, ":", 3)
	if len(parts) < 2 {
		return sub
	}
	return parts[0] + ":" + parts[1]
}

// expectedSubjects are the Subject patterns clog expects for a repo, built from
// the prefix the platform really sends. They match `clog CI --config-help`.
func expectedSubjects(platform Platform, prefix string) (dev, prod string) {
	if platform == PlatformGitLab {
		return prefix + ":ref_type:*:ref:*", prefix + ":ref_type:tag:ref:v*"
	}
	return prefix + ":ref:refs/*", prefix + ":ref:refs/tags/v*"
}

// expandBraces expands bash-style brace lists: "a{b,c}d" -> [abd acd]. A brace
// group with no top-level comma is literal, as it is in Infisical.
func expandBraces(s string) []string {
	start, depth := -1, 0
	for i, r := range s {
		switch r {
		case '{':
			if depth == 0 {
				start = i
			}
			depth++
		case '}':
			if depth == 0 {
				continue
			}
			depth--
			if depth == 0 {
				alts := splitTopLevel(s[start+1 : i])
				if len(alts) < 2 {
					// literal group: keep it, expand what follows
					var out []string
					for _, rest := range expandBraces(s[i+1:]) {
						out = append(out, s[:i+1]+rest)
					}
					return out
				}
				var out []string
				for _, a := range alts {
					out = append(out, expandBraces(s[:start]+a+s[i+1:])...)
				}
				return out
			}
		}
	}
	return []string{s}
}

// splitTopLevel splits s on commas that are not inside braces.
func splitTopLevel(s string) []string {
	var out []string
	depth, last := 0, 0
	for i, r := range s {
		switch r {
		case '{':
			depth++
		case '}':
			depth--
		case ',':
			if depth == 0 {
				out = append(out, s[last:i])
				last = i + 1
			}
		}
	}
	return append(out, s[last:])
}

// globMatch reports whether subject matches pattern the way Infisical matches
// a Subject: exactly, or as a glob where * also matches "/" and {a,b} lists
// alternatives.
func globMatch(pattern, subject string) bool {
	for _, alt := range expandBraces(pattern) {
		var b strings.Builder
		b.WriteString("^")
		for _, r := range alt {
			switch r {
			case '*':
				b.WriteString(".*")
			case '?':
				b.WriteString(".")
			default:
				b.WriteString(regexp.QuoteMeta(string(r)))
			}
		}
		b.WriteString("$")
		if regexp.MustCompile(b.String()).MatchString(subject) {
			return true
		}
	}
	return false
}

// concrete turns a pattern prefix into one real-looking string: the first
// brace alternative, with each wildcard replaced by "x".
func concrete(p string) string {
	return strings.NewReplacer("*", "x", "?", "x").Replace(expandBraces(p)[0])
}

// lintSubject lists the reasons a bound Subject pattern is wrong or unsafe.
// repo is "<owner>/<name>" when known. An empty result means it looks right.
func lintSubject(pattern string, platform Platform, mode, repo string) []string {
	var out []string
	if pattern == "" {
		return []string{"no Subject is bound: every token is refused, or (if Infisical skips an empty Subject) every repo is admitted"}
	}
	if strings.TrimSpace(pattern) != pattern {
		out = append(out, "starts or ends with whitespace, so it never matches: delete the space")
	} else if strings.ContainsAny(pattern, " \t\n") {
		out = append(out, "contains whitespace, which no subject has")
	}
	for _, m := range regexp.MustCompile(`\{[^{},]*\}`).FindAllString(pattern, -1) {
		out = append(out, fmt.Sprintf("%s is matched literally: a brace list needs a comma, so write the value without braces", m))
	}
	if len(splitTopLevel(pattern)) > 1 {
		out = append(out, "has a comma outside braces: bind one pattern, and put alternatives in one {a,b} list")
	}
	head, tail := "repo:", ":ref:"
	branchProbe := ":ref:refs/heads/zz-clog-probe"
	if platform == PlatformGitLab {
		head, tail = "project_path:", ":ref_type:"
		branchProbe = ":ref_type:branch:ref:zz-clog-probe"
	}
	if !strings.HasPrefix(pattern, head) {
		out = append(out, fmt.Sprintf("does not start with %q, the form %s sends", head, platform))
	}
	prefix := subjectPrefix(pattern)
	if strings.Contains(prefix, "*") || strings.Contains(prefix, "?") {
		out = append(out, fmt.Sprintf("has a wildcard in the repo part (%s): it admits other repos", prefix))
	}
	if _, name, ok := strings.Cut(repo, "/"); ok && name != "" && !namesRepo(prefix, name) {
		out = append(out, fmt.Sprintf("does not name this repo (%s)", repo))
	}
	if mode == ModeProd {
		probe := concrete(prefix) + branchProbe
		// a GitHub Environment subject (repo:<o>/<r>:environment:prod) is gated by
		// the Environment's own protection rules, not by a ref
		bound := strings.Contains(pattern, tail) || strings.Contains(pattern, ":environment:")
		if !bound || globMatch(pattern, probe) {
			out = append(out, "admits any branch, so a feature-branch push gets prod secrets: bind release tags only (clog CI --config-help)")
		}
	}
	return out
}

// namesRepo reports whether any brace alternative of a subject prefix is for
// repo name: ".../<name>" or, for a GitHub immutable subject, ".../<name>@<id>".
func namesRepo(prefix, name string) bool {
	for _, alt := range expandBraces(prefix) {
		_, last, _ := strings.Cut(alt, "/")
		last, _, _ = strings.Cut(last, "@")
		if last == name {
			return true
		}
	}
	return false
}

// identityOIDC is the part of Infisical's identity OIDC config that is checked.
type identityOIDC struct {
	BoundSubject   string `json:"boundSubject"`
	BoundAudiences string `json:"boundAudiences"`
	BoundIssuer    string `json:"boundIssuer"`
}

// fetchIdentityOIDC reads an identity's OIDC auth config with a user token
// (GET /api/v1/auth/oidc-auth/identities/<uuid>).
func fetchIdentityOIDC(domain, token, uuid string) (identityOIDC, error) {
	req, err := http.NewRequest(http.MethodGet, strings.TrimRight(domain, "/")+"/api/v1/auth/oidc-auth/identities/"+uuid, nil)
	if err != nil {
		return identityOIDC{}, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	var out struct {
		IdentityOidcAuth identityOIDC `json:"identityOidcAuth"`
	}
	if err := doJSON(req, &out); err != nil {
		return identityOIDC{}, fmt.Errorf("read OIDC auth of identity %s: %w", uuid, err)
	}
	return out.IdentityOidcAuth, nil
}

var oidcCmd = &cobra.Command{
	Use:   "oidc",
	Short: "show the OIDC subject CI presents to Infisical, and check the bound Subjects",
	Long: `clog CI oidc - make an Infisical "OIDC subject not allowed" (403) explain itself.

In a CI job it prints the claims this job's token presents (never the token) and
the Subject patterns to bind for this repo, built from the subject the platform
really sends (so a GitHub immutable subject, repo:<owner>@<id>/<repo>@<id>, comes
out right).

On a laptop, after  infisical login --domain=<ci.infisical.domain> , it reads
the Subject bound to every identity in ci.infisical.identity and checks it:
whitespace, one-item braces, top-level commas, wildcards in the repo part, a
pattern that does not name this repo, and a prod identity that admits branches.
It exits 1 when any check fails.`,
	SilenceUsage: true,
	Args:         cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runOIDC(DefaultEnv(), cmd.OutOrStdout())
	},
}

func init() { Command.AddCommand(oidcCmd) }

// runOIDC is `clog CI oidc`.
func runOIDC(env Env, w io.Writer) error {
	cfg, err := LoadConfig()
	if err != nil {
		return err
	}
	inf := cfg.Infisical
	if inf.isUnset() {
		return fmt.Errorf("no ci.infisical block in .clog.yaml: nothing to check (clog CI --config-help)")
	}
	if err := inf.validate(); err != nil {
		return err
	}
	r, err := Resolve(env)
	if err != nil {
		return err
	}
	if r.CI == PlatformGitHub || r.CI == PlatformGitLab {
		return oidcInCI(env, r, inf, w)
	}
	return oidcOnLaptop(r, inf, w)
}

// oidcInCI prints this job's claims and the patterns to bind.
func oidcInCI(env Env, r Resolution, inf InfisicalConfig, w io.Writer) error {
	printf := func(format string, a ...any) { _, _ = fmt.Fprintf(w, format, a...) }
	jwt, err := platformJWT(env, r.CI, inf.audience())
	if err != nil {
		return err
	}
	c, err := tokenClaims(jwt)
	if err != nil {
		return err
	}
	printf("this %s job presents:\n", r.CI)
	for _, k := range claimKeys {
		if v, ok := c[k]; ok {
			printf("  %-17s %v\n", k, v)
		}
	}
	sub, _ := c["sub"].(string)
	dev, prod := expectedSubjects(r.CI, subjectPrefix(sub))
	printf("bind these Subjects (Audiences: %s):\n  dev-uuid   %s\n  prod-uuid  %s\n", inf.audience(), dev, prod)
	if d, err := Decide(env); err == nil {
		key, uuid, _ := inf.identityFor(r.CI, d.Mode)
		printf("this run is %s, so it logs in as %s %s\n", d.Mode, key, uuid)
	}
	return nil
}

// oidcOnLaptop reads every configured identity's bound Subject and lints it.
func oidcOnLaptop(r Resolution, inf InfisicalConfig, w io.Writer) error {
	printf := func(format string, a ...any) { _, _ = fmt.Fprintf(w, format, a...) }
	token, err := userToken(inf.Domain)
	if err != nil {
		return err
	}
	platforms := make([]string, 0, len(inf.Identity))
	for p := range inf.Identity {
		platforms = append(platforms, p)
	}
	sort.Strings(platforms)
	failed := 0
	for _, p := range platforms {
		for _, mode := range []string{ModeDev, ModeProd} {
			key, uuid, err := inf.identityFor(Platform(p), mode)
			if err != nil {
				continue // a platform this repo does not build on
			}
			got, err := fetchIdentityOIDC(inf.Domain, token, uuid)
			if err != nil {
				printf("FAIL %s %s %s: %v\n", p, key, uuid, err)
				failed++
				continue
			}
			problems := lintSubject(got.BoundSubject, Platform(p), mode, r.Repo)
			if strings.TrimSpace(got.BoundAudiences) == "" {
				problems = append(problems, "Audiences is blank, so the audience check is skipped: set it to "+inf.audience())
			}
			status := "ok  "
			if len(problems) > 0 {
				status = "FAIL"
				failed++
			}
			printf("%s %s %s %s\n     Subject %q\n", status, p, key, uuid, got.BoundSubject)
			for _, msg := range problems {
				printf("     - %s\n", msg)
			}
		}
	}
	if failed > 0 {
		return fmt.Errorf("%d identity check(s) failed: fix them in Infisical (Access Control -> Identities -> <name> -> Authentication -> OIDC Auth)", failed)
	}
	return nil
}
