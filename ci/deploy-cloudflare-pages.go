//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/

package ci

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
)

// Cloudflare Pages target data, per mode:
//
//	pages:
//	  kind: cloudflare-pages
//	  require: [CLOUDFLARE_API_TOKEN, CLOUDFLARE_ACCOUNT_ID]
//	  dev:  {project: site-staging, domain: staging.example.org, production-branch: main, dir: public}
//	  prod: {project: site,         domain: example.org,         production-branch: main, dir: public}
//
//	dir                directory to upload                    (required)
//	project            Pages project name                     (required)
//	production-branch  the project's production branch        (required)
//	domain             the site's domain, for the log line    (optional)
//
// Two separate Pages PROJECTS are the staging mechanism, not two branches of
// one project: which project you upload to decides which domain serves the
// result. So --branch is always the project's own production branch, whatever
// git branch the build came from - a dev build uploaded to the dev project is
// a production deployment of that project. The real branch travels in the
// commit message, so the Cloudflare dashboard still says where it came from.
//
// The project must already exist. Creating one is an account-level change with
// a DNS side effect, and a deployer that does that silently is how a live
// domain moves without anyone deciding to move it.
func deployCloudflarePages(d Deployment) error {
	dir, err := d.Get("dir", true)
	if err != nil {
		return err
	}
	project, err := d.Get("project", true)
	if err != nil {
		return err
	}
	prodBranch, err := d.Get("production-branch", true)
	if err != nil {
		return err
	}
	domain, err := d.Get("domain", false)
	if err != nil {
		return err
	}

	if err := checkPublishDir(dir); err != nil {
		return err
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("cannot resolve %s: %w", dir, err)
	}

	for _, k := range []string{"CLOUDFLARE_API_TOKEN", "CLOUDFLARE_ACCOUNT_ID"} {
		if strings.TrimSpace(d.Env.Getenv(k)) == "" {
			return fmt.Errorf("$%s is not set: run this under `clog CI run` so the secret store supplies it", k)
		}
	}

	version := ReleaseVersion()
	branch := gitBranch(d.Env)
	msg := fmt.Sprintf("%s (%s from %s)", version, d.Mode, branch)

	slog.Info("cloudflare-pages", "project", project, "dir", abs, "branch", prodBranch,
		"version", version, "from", branch)

	if d.DryRun {
		fmt.Fprintf(d.Out, "dry-run: would upload %s/ to Pages project %s on %s (%s)\n",
			dir, project, prodBranch, version)
		return nil
	}

	args := []string{
		"wrangler", "pages", "deploy", abs,
		"--project-name=" + project,
		"--branch=" + prodBranch,
		"--commit-dirty=true",
		"--commit-message=" + msg,
	}
	if sha := commitSHA(d.Env); sha != "" {
		args = append(args, "--commit-hash="+sha)
	}

	if out, err := execCommand(".", "npx", args...); err != nil {
		return fmt.Errorf("wrangler pages deploy: %w%s", err, indentOutput(out))
	}

	where := project
	if domain != "" {
		where = "https://" + domain
	}
	fmt.Fprintf(d.Out, "deployed %s to %s\n", version, where)
	return nil
}

// gitBranch names the branch being built, falling back to whatever the CI
// platform says when the checkout is detached - which it is for a tag build.
func gitBranch(env Env) string {
	if b := firstNonEmpty(env.Getenv("CI_COMMIT_REF_NAME"), env.Getenv("GITHUB_REF_NAME")); b != "" {
		return b
	}
	if out, err := execCommand(".", "git", "branch", "--show-current"); err == nil && out != "" {
		return out
	}
	return "detached"
}
