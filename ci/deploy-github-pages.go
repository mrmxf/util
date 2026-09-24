//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/

package ci

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// GitHub Pages target data, per mode:
//
//	pages:
//	  kind: github-pages
//	  prod: {dir: kodata, branch: gh-pages, cname: example.org}
//
//	dir     directory to publish            (required)
//	branch  orphan branch to force-push to  (default gh-pages)
//	repo    owner/name to push to           (default: this repo's origin)
//	cname   custom domain, written as CNAME (optional)
//
// Pages must already be pointed at the branch. Turning Pages on is a one-time
// account-level change and this deployer will not do it silently: a repo with
// Pages unconfigured gets a published branch and a clear message, not a
// surprise change to a live site's serving source.
//
// Pages pointed somewhere ELSE is refused before anything is pushed. A push to a
// branch nothing serves succeeds, and the site does not change - pihuw shipped
// five releases that way. Reading the setting needs the token to have
// `pages: read`; without it the deploy warns and carries on.
const (
	pagesDefaultBranch = "gh-pages"
	pagesBotName       = "clog"
	pagesBotEmail      = "clog@mrmxf.com"
)

func deployGitHubPages(d Deployment) error {
	dir, err := d.Get("dir", true)
	if err != nil {
		return err
	}
	branch, err := d.GetOr("branch", pagesDefaultBranch)
	if err != nil {
		return err
	}
	repo, err := d.Get("repo", false)
	if err != nil {
		return err
	}
	if repo == "" {
		if repo, err = repoSlug(d.Env); err != nil {
			return err
		}
	}
	cname, err := d.Get("cname", false)
	if err != nil {
		return err
	}
	if repo == "" {
		return fmt.Errorf("cannot work out which repo to push to: set %s.%s.repo or add an origin remote",
			d.Target, d.Mode)
	}
	if err := checkPublishDir(dir); err != nil {
		return err
	}
	// dir is usually relative in config ("kodata"). Everything below passes it
	// to git as --work-tree while also running git inside it, so a relative
	// path would be resolved twice - git would look for kodata/kodata.
	dir, err = filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("cannot resolve %s: %w", dir, err)
	}

	version := ReleaseVersion()
	push, redacted := pagesPushURL(d.Env, repo)
	slog.Info("github-pages", "repo", repo, "branch", branch, "dir", dir, "version", version, "push", redacted)

	if d.DryRun {
		fmt.Fprintf(d.Out, "dry-run: would publish %s/ to %s on %s (%s)\n", dir, repo, branch, version)
		return nil
	}

	enabled, err := checkPagesSource(d, repo, branch)
	if err != nil {
		return err
	}

	if err := writeReleaseMarker(dir, version); err != nil {
		return err
	}
	if cname != "" {
		if err := os.WriteFile(filepath.Join(dir, "CNAME"), []byte(cname+"\n"), 0o644); err != nil {
			return fmt.Errorf("cannot write CNAME: %w", err)
		}
	}
	// Pages serves the branch verbatim; without .nojekyll GitHub runs Jekyll,
	// which silently drops files and directories beginning with an underscore.
	if err := os.WriteFile(filepath.Join(dir, ".nojekyll"), nil, 0o644); err != nil {
		return fmt.Errorf("cannot write .nojekyll: %w", err)
	}

	// Build the commit in a throwaway git dir with dir as the work tree, so the
	// published directory never gains a .git of its own and the repo's own index
	// is untouched.
	gitDir, err := mkdirTemp("", "clog-pages-")
	if err != nil {
		return fmt.Errorf("cannot create a temporary git dir: %w", err)
	}
	defer os.RemoveAll(gitDir)

	msg := fmt.Sprintf("deploy %s to %s", version, branch)
	steps := [][]string{
		{"init", "-q", "-b", branch},
		// `git init --git-dir=X` marks the repo bare, and every later command
		// then refuses with "this operation must be run in a work tree" however
		// much --work-tree is passed. Clear it before touching the index.
		{"config", "core.bare", "false"},
		{"add", "-A"},
		{"-c", "user.name=" + pagesBotName, "-c", "user.email=" + pagesBotEmail,
			"commit", "-q", "--allow-empty", "-m", msg},
		{"push", "--force", "-q", push, branch},
	}
	for _, step := range steps {
		args := append([]string{"--git-dir=" + gitDir, "--work-tree=" + dir}, step...)
		if out, err := execCommand(dir, "git", args...); err != nil {
			return fmt.Errorf("git %s: %w%s", step[0], err, indentOutput(redact(out, push, redacted)))
		}
	}
	fmt.Fprintf(d.Out, "published %s/ to %s on %s (%s)\n", dir, repo, branch, version)
	if !enabled {
		fmt.Fprintf(d.Out, "Pages is not enabled for %s, or this token cannot read its settings: "+
			"the branch is published but nothing serves it until Settings → Pages → "+
			"Deploy from a branch → %s, / (root)\n", repo, branch)
	}
	return nil
}

// pagesInfo is the part of GET /repos/{repo}/pages the deployer cares about.
type pagesInfo struct {
	BuildType string `json:"build_type"` // "legacy" serves a branch; "workflow" does not
	Source    struct {
		Branch string `json:"branch"`
		Path   string `json:"path"`
	} `json:"source"`
}

// pagesSettings reads a repo's Pages configuration. A seam of its own, so a
// test that runs real git does not also reach the GitHub API.
var pagesSettings = func(repo string) (string, error) {
	return execCommand(".", "gh", "api", "repos/"+repo+"/pages")
}

// checkPagesSource refuses a publish that could not change the live site. It
// reports whether Pages is known to be enabled; a lookup that fails for any
// other reason is a warning, not an error, since the push may still be right.
func checkPagesSource(d Deployment, repo, branch string) (enabled bool, err error) {
	out, err := pagesSettings(repo)
	if err != nil {
		if strings.Contains(out, "HTTP 404") {
			return false, nil
		}
		slog.Warn("github-pages: cannot read which branch Pages serves; publishing anyway",
			"repo", repo, "err", err, "out", out)
		return true, nil
	}
	var p pagesInfo
	if err := json.Unmarshal([]byte(out), &p); err != nil {
		slog.Warn("github-pages: cannot parse the Pages settings; publishing anyway", "repo", repo, "err", err)
		return true, nil
	}
	fix := fmt.Sprintf("Settings → Pages → Source: Deploy from a branch → %s, / (root). "+
		"clog will not change a live site's serving source for you", branch)
	if p.BuildType == "workflow" {
		return true, fmt.Errorf("the Pages site for %s is built by a GitHub Actions workflow, not served from %s: "+
			"publishing would change nothing live. %s", repo, branch, fix)
	}
	if p.Source.Branch != branch || (p.Source.Path != "" && p.Source.Path != "/") {
		return true, fmt.Errorf("the Pages site for %s serves %s:%s, not %s:/ - publishing would change nothing live. %s",
			repo, p.Source.Branch, p.Source.Path, branch, fix)
	}
	return true, nil
}

// checkPublishDir fails early and specifically: an empty directory here means
// the build step did not run, and force-pushing it would blank a live site.
func checkPublishDir(dir string) error {
	info, err := os.Stat(dir)
	if err != nil {
		return fmt.Errorf("cannot publish %s: %w (has the build step run?)", dir, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("cannot publish %s: not a directory", dir)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("cannot read %s: %w", dir, err)
	}
	if len(entries) == 0 {
		return fmt.Errorf("refusing to publish %s: it is empty, which would blank the site", dir)
	}
	return nil
}

// pagesPushURL returns the URL to push to and a safe form for logging.
//
// In CI the token is the only credential available, so it goes in the URL. The
// redacted form is what gets logged and what error output is scrubbed with, so
// the token never reaches a job log. On a laptop with no token the origin
// remote is used as-is, which picks up the developer's ssh key.
func pagesPushURL(env Env, repo string) (push, redacted string) {
	if token := pagesToken(env); token != "" {
		return fmt.Sprintf("https://x-access-token:%s@github.com/%s.git", token, repo),
			fmt.Sprintf("https://x-access-token:***@github.com/%s.git", repo)
	}
	if g, ok := env.Git.(interface{ RemoteURL() string }); ok {
		if url := g.RemoteURL(); url != "" {
			return url, url
		}
	}
	url := "https://github.com/" + repo + ".git"
	return url, url
}

func pagesToken(env Env) string {
	return firstNonEmpty(env.Getenv("GH_TOKEN"), env.Getenv("GITHUB_TOKEN"))
}

// repoSlug is the owner/name this deploy publishes to.
//
// Resolve already normalises GITHUB_REPOSITORY / CI_PROJECT_PATH / the origin
// remote into Resolution.Repo, so this reuses it rather than reading the same
// variables a second time. The one thing it must not inherit is a pull request:
// there Resolve reports the *fork* as the repo, and publishing a fork's build
// to the base repo's Pages branch would let any PR author overwrite the live
// site. Policy stops a PR deploying long before this point, so it is a guard
// rather than a workflow.
func repoSlug(env Env) (string, error) {
	r, err := Resolve(env)
	if err != nil {
		return "", err
	}
	if r.Verb == VerbPR {
		return "", fmt.Errorf("refusing to publish a pull-request build: it would push fork content to the live Pages branch")
	}
	return r.Repo, nil
}

// redact replaces any credential-bearing URL in command output.
func redact(out, secret, safe string) string {
	if secret == "" || secret == safe {
		return out
	}
	return strings.ReplaceAll(out, secret, safe)
}

func indentOutput(out string) string {
	if strings.TrimSpace(out) == "" {
		return ""
	}
	return "\n  " + strings.ReplaceAll(out, "\n", "\n  ")
}
