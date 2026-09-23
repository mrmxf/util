//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/

package ci

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// pagesConfig is a site that publishes one directory to GitHub Pages in prod.
func pagesConfig(prod map[string]any) Config {
	return Config{Targets: map[string]Target{
		"pages": {Kind: KindGitHubPages, Modes: []string{ModeProd}, Prod: prod},
	}}
}

// recordExec swaps execCommand for a recorder and returns the captured calls
// plus a restore func.
func recordExec(t *testing.T, fail string) (*[]string, func()) {
	t.Helper()
	var calls []string
	prev := execCommand
	execCommand = func(dir, name string, args ...string) (string, error) {
		calls = append(calls, name+" "+strings.Join(args, " "))
		if fail != "" && strings.Contains(strings.Join(args, " "), fail) {
			// git leaks the push URL in its errors; the deployer must scrub it
			return "fatal: could not read from https://x-access-token:s3cr3t@github.com/acme/site.git", errors.New("exit 128")
		}
		return "", nil
	}
	return &calls, func() { execCommand = prev }
}

// builtSite makes a non-empty directory standing in for a build output.
func builtSite(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("<h1>hi</h1>"), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func pagesEnv(vars map[string]string) Env {
	return laptop(vars, fakeGit{slug: "acme/site", url: "git@github.com:acme/site.git"})
}

func TestDeployGitHubPagesPublishes(t *testing.T) {
	dir := builtSite(t)
	calls, restore := recordExec(t, "")
	defer restore()
	prevVer := ReleaseVersion
	ReleaseVersion = func() string { return "v1.2.3" }
	defer func() { ReleaseVersion = prevVer }()

	var out bytes.Buffer
	cfg := pagesConfig(map[string]any{"dir": dir, "cname": "example.org"})
	if err := Deploy(pagesEnv(map[string]string{"GITHUB_TOKEN": "s3cr3t"}), cfg, ModeProd, "", false, &out); err != nil {
		t.Fatalf("deploy: %v", err)
	}

	joined := strings.Join(*calls, "\n")
	for _, want := range []string{"git --git-dir=", "init -q -b gh-pages", "add -A", "commit", "push --force"} {
		if !strings.Contains(joined, want) {
			t.Errorf("expected a %q step, got:\n%s", want, joined)
		}
	}
	// the token must be used for the push...
	if !strings.Contains(joined, "https://x-access-token:s3cr3t@github.com/acme/site.git") {
		t.Errorf("push should authenticate with the token, got:\n%s", joined)
	}
	// ...but never appear in what the user sees
	if strings.Contains(out.String(), "s3cr3t") {
		t.Errorf("token leaked into output: %s", out.String())
	}
	// Pages runs Jekyll unless told not to, which would drop _ files
	if _, err := os.Stat(filepath.Join(dir, ".nojekyll")); err != nil {
		t.Errorf(".nojekyll should be written: %v", err)
	}
	cname, err := os.ReadFile(filepath.Join(dir, "CNAME"))
	if err != nil || strings.TrimSpace(string(cname)) != "example.org" {
		t.Errorf("CNAME = %q, %v", cname, err)
	}
	if !strings.Contains(out.String(), "v1.2.3") {
		t.Errorf("output should name the version, got %q", out.String())
	}
}

func TestDeployGitHubPagesRefusesEmptyDir(t *testing.T) {
	calls, restore := recordExec(t, "")
	defer restore()
	cfg := pagesConfig(map[string]any{"dir": t.TempDir()})
	err := Deploy(pagesEnv(nil), cfg, ModeProd, "", false, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "would blank the site") {
		t.Fatalf("an empty publish dir must be refused, got %v", err)
	}
	if len(*calls) != 0 {
		t.Errorf("nothing should run for an empty dir, got %v", *calls)
	}
}

func TestDeployGitHubPagesMissingDirIsNamed(t *testing.T) {
	cfg := pagesConfig(map[string]any{"branch": "gh-pages"}) // no dir
	err := Deploy(pagesEnv(nil), cfg, ModeProd, "", false, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "dir is not set") {
		t.Fatalf("a missing dir must be named, got %v", err)
	}
}

func TestDeployGitHubPagesDryRunTouchesNothing(t *testing.T) {
	dir := builtSite(t)
	calls, restore := recordExec(t, "")
	defer restore()
	var out bytes.Buffer
	cfg := pagesConfig(map[string]any{"dir": dir})
	if err := Deploy(pagesEnv(nil), cfg, ModeProd, "", true, &out); err != nil {
		t.Fatalf("dry run: %v", err)
	}
	if len(*calls) != 0 {
		t.Errorf("dry run must not run commands, got %v", *calls)
	}
	if _, err := os.Stat(filepath.Join(dir, ".nojekyll")); err == nil {
		t.Error("dry run must not write .nojekyll")
	}
	if !strings.Contains(out.String(), "dry-run") {
		t.Errorf("dry run should say so, got %q", out.String())
	}
}

func TestDeployGitHubPagesRedactsTokenOnFailure(t *testing.T) {
	dir := builtSite(t)
	_, restore := recordExec(t, "push")
	defer restore()
	cfg := pagesConfig(map[string]any{"dir": dir})
	err := Deploy(pagesEnv(map[string]string{"GH_TOKEN": "s3cr3t"}), cfg, ModeProd, "", false, &bytes.Buffer{})
	if err == nil {
		t.Fatal("a failed push must be an error")
	}
	if strings.Contains(err.Error(), "s3cr3t") {
		t.Errorf("token leaked into the error: %v", err)
	}
	if !strings.Contains(err.Error(), "x-access-token:***") {
		t.Errorf("error should keep the redacted URL for context: %v", err)
	}
}

func TestDeployUnimplementedKindIsAnError(t *testing.T) {
	cfg := Config{Targets: map[string]Target{
		"cdn": {Kind: KindBucket, Modes: []string{ModeProd}, Prod: map[string]any{"bucket": "x"}},
	}}
	err := Deploy(pagesEnv(nil), cfg, ModeProd, "", false, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "no deployer yet") {
		t.Fatalf("a declared kind with no deployer must say so, got %v", err)
	}
}

func TestDeployTargetFlag(t *testing.T) {
	dir := builtSite(t)
	cfg := pagesConfig(map[string]any{"dir": dir})
	cfg.Targets["cdn"] = Target{Kind: KindBucket, Modes: []string{ModeProd}, Prod: map[string]any{"bucket": "x"}}

	_, restore := recordExec(t, "")
	defer restore()
	// picking the implemented target must not touch the unimplemented one
	if err := Deploy(pagesEnv(nil), cfg, ModeProd, "pages", false, &bytes.Buffer{}); err != nil {
		t.Fatalf("--target pages: %v", err)
	}
	if err := Deploy(pagesEnv(nil), cfg, ModeProd, "nope", false, &bytes.Buffer{}); err == nil ||
		!strings.Contains(err.Error(), "no ci.targets.nope") {
		t.Fatalf("an unknown target must be named, got %v", err)
	}
	if err := Deploy(pagesEnv(nil), cfg, ModeDev, "pages", false, &bytes.Buffer{}); err == nil ||
		!strings.Contains(err.Error(), "does not deploy in dev") {
		t.Fatalf("a target not used in the mode must say so, got %v", err)
	}
}

func TestPagesPushURLFallsBackToOrigin(t *testing.T) {
	push, redacted := pagesPushURL(pagesEnv(nil), "acme/site")
	if push != "git@github.com:acme/site.git" || redacted != push {
		t.Errorf("with no token expected the origin remote, got %q / %q", push, redacted)
	}
}

func TestRepoSlugReusesResolve(t *testing.T) {
	// no CI vars -> local resolution -> the origin remote's slug
	got, err := repoSlug(pagesEnv(nil))
	if err != nil || got != "acme/site" {
		t.Errorf("repoSlug = %q, %v; want acme/site", got, err)
	}
}

// A pull-request build must never be published: Resolve reports the fork as the
// repo, so inheriting it would push fork content to the live Pages branch.
func TestDeployGitHubPagesRefusesPullRequest(t *testing.T) {
	dir := builtSite(t)
	calls, restore := recordExec(t, "")
	defer restore()
	env := fakeEnv(map[string]string{
		"GITHUB_ACTIONS":    "true",
		"GITHUB_EVENT_NAME": "pull_request",
		"GITHUB_REPOSITORY": "acme/site",
		"GITHUB_REF":        "refs/pull/7/merge",
	}, `{"pull_request": {"head": {"sha": "abc", "repo": {"full_name": "fork/site", "html_url": "https://github.com/fork/site"}}}, "sender": {"login": "someone"}}`)

	cfg := pagesConfig(map[string]any{"dir": dir})
	err := Deploy(env, cfg, ModeProd, "", false, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "pull-request build") {
		t.Fatalf("a PR build must be refused, got %v", err)
	}
	if len(*calls) != 0 {
		t.Errorf("nothing should run for a PR, got %v", *calls)
	}
}

// TestDeployGitHubPagesAgainstRealGit exercises the git steps for real against
// a local bare repo. The stubbed tests above cannot catch a wrong flag: the
// --git-dir form marks the repo bare, and every command after init then fails.
func TestDeployGitHubPagesAgainstRealGit(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	origin := filepath.Join(t.TempDir(), "origin.git")
	if out, err := exec.Command("git", "init", "-q", "--bare", origin).CombinedOutput(); err != nil {
		t.Fatalf("bare init: %v %s", err, out)
	}
	// Config gives a path relative to the repo root ("kodata"), not an absolute
	// one, so run from a working directory and publish a relative dir.
	work := t.TempDir()
	if err := os.MkdirAll(filepath.Join(work, "kodata"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(work, "kodata", "index.html"), []byte("<h1>hi</h1>"), 0o644); err != nil {
		t.Fatal(err)
	}
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(work); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(cwd) //nolint:errcheck // test cleanup
	dir := "kodata"

	prevVer := ReleaseVersion
	ReleaseVersion = func() string { return "v9.9.9" }
	defer func() { ReleaseVersion = prevVer }()

	env := laptop(nil, fakeGit{slug: "acme/site", url: origin})
	cfg := pagesConfig(map[string]any{"dir": dir, "branch": "gh-pages"})
	if err := Deploy(env, cfg, ModeProd, "", false, &bytes.Buffer{}); err != nil {
		t.Fatalf("deploy: %v", err)
	}

	files, err := exec.Command("git", "--git-dir="+origin, "ls-tree", "-r", "--name-only", "gh-pages").Output()
	if err != nil {
		t.Fatalf("nothing was pushed: %v", err)
	}
	got := strings.Fields(string(files))
	for _, want := range []string{"index.html", ".nojekyll"} {
		if !slices.Contains(got, want) {
			t.Errorf("gh-pages should contain %s, has %v", want, got)
		}
	}
	// the published directory must not gain a .git of its own
	if _, err := os.Stat(filepath.Join(dir, ".git")); !os.IsNotExist(err) {
		t.Errorf("publish dir should stay clean, .git exists")
	}
}

// `clog deploy prod` must actually deploy in prod. Sites used to carry a private
// bc-mode snippet to turn an argument into CLOG_MODE; --mode does it here so the
// copies can go.
func TestDeployModeFlagForcesTargetSelection(t *testing.T) {
	dir := builtSite(t)
	cfg := pagesConfig(map[string]any{"dir": dir}) // prod-only target
	calls, restore := recordExec(t, "")
	defer restore()

	// the resolved mode is dev, where this target does not deploy
	if err := Deploy(pagesEnv(nil), cfg, ModeDev, "", false, &bytes.Buffer{}); err != nil {
		t.Fatalf("dev: %v", err)
	}
	if len(*calls) != 0 {
		t.Errorf("a prod-only target must not deploy in dev, got %v", *calls)
	}
	// forced to prod, it does
	if err := Deploy(pagesEnv(nil), cfg, ModeProd, "", false, &bytes.Buffer{}); err != nil {
		t.Fatalf("prod: %v", err)
	}
	if len(*calls) == 0 {
		t.Error("forcing prod should have published")
	}
}

// A repo with no secret store must still run. Not every repo has secrets - a
// docs site publishing to GitHub Pages needs only the token Actions provides -
// and `clog CI run` used to fail such a repo with "missing .clog.yaml keys".
func TestInfisicalIsUnset(t *testing.T) {
	if !(InfisicalConfig{}).isUnset() {
		t.Error("an absent ci.infisical block should be unset")
	}
	// a partially filled block is a typo, not an opt-out
	partial := InfisicalConfig{Domain: "https://eu.infisical.com"}
	if partial.isUnset() {
		t.Error("a partial ci.infisical block must NOT count as unset")
	}
	if err := partial.validate(); err == nil {
		t.Error("a partial ci.infisical block must still fail validation")
	}
}
