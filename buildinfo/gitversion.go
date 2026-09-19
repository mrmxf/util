//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

package buildinfo

// Build information from the state of the git repository, not releases.yaml.
// releases.yaml is history; git tags are the instruction.
//
//	HEAD exactly on a release tag       v0.11.12
//	3 commits after it                  v0.11.12+dev.3.g34103be
//	...with uncommitted changes         v0.11.12+dev.3.g34103be.dirty
//	no release tag reachable            v0.0.0+dev.41.g34103be
//
// A release tag matches ReleaseTagRe: v1.2.3 or 1.2.3, nothing after it, so
// v3.5.0-dev or v1.0.0-rc1 never count as releases. The +... part is SemVer
// build metadata: it sorts equal to the release it follows and is valid for Go
// ldflags, npm and PEP 440 (as a local version), but NOT for Docker tags
// (use DockerTag) or git tag names.

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ReleaseTagRe matches a release tag: optional v, three numbers, nothing else.
var ReleaseTagRe = regexp.MustCompile(`^v?[0-9]+\.[0-9]+\.[0-9]+$`)

// IsReleaseTag reports whether name is a release tag (see ReleaseTagRe).
func IsReleaseTag(name string) bool { return ReleaseTagRe.MatchString(name) }

// GitState is what a build needs to know about the working copy.
type GitState struct {
	Hash       string // full HEAD commit id
	CommitDate string // YYYY-MM-DD of HEAD (reproducible, unlike today's date)
	Dirty      bool   // uncommitted changes to tracked or untracked-not-ignored files
	ExactTag   string // release tag pointing at HEAD, "" if none
	Nearest    string // newest release tag reachable from HEAD, "" if none
	Ahead      int    // commits from Nearest (or the root) to HEAD
}

// Short is the abbreviated commit id used in dev versions.
func (g GitState) Short() string {
	if len(g.Hash) >= 7 {
		return g.Hash[:7]
	}
	return g.Hash
}

// Version is the build's version string (see the file comment).
func (g GitState) Version() string {
	dirty := ""
	if g.Dirty {
		dirty = ".dirty"
	}
	if g.ExactTag != "" {
		if dirty == "" {
			return g.ExactTag
		}
		return g.ExactTag + "+dirty"
	}
	base := g.Nearest
	if base == "" {
		base = "v0.0.0"
	}
	return fmt.Sprintf("%s+dev.%d.g%s%s", base, g.Ahead, g.Short(), dirty)
}

// DockerTag is Version with the characters OCI tags forbid replaced: + → -.
func DockerTag(version string) string { return strings.ReplaceAll(version, "+", "-") }

// BuildOptions are the parts of the linker data that do not come from git.
type BuildOptions struct {
	Mode   string // dev | prod
	Name   string // command name, e.g. clog
	Title  string // printable name; spaces become _
	Branch string // dev builds: shown as the version suffix
}

// Generate makes the linker data for a build. prod is refused unless HEAD is
// exactly on a release tag and the tree is clean: a production binary must be
// reproducible from a tag.
func Generate(g GitState, o BuildOptions) (LinkerDataJSON, error) {
	switch o.Mode {
	case "", "dev":
		o.Mode = "dev"
	case "prod":
		if g.ExactTag == "" {
			near := "no release tag is reachable"
			if g.Nearest != "" {
				near = fmt.Sprintf("HEAD is %d commits after %s", g.Ahead, g.Nearest)
			}
			return LinkerDataJSON{}, fmt.Errorf("prod build needs HEAD on a release tag (vX.Y.Z): %s - check out the tag, e.g. clog BC git checkout production", near)
		}
		if g.Dirty {
			return LinkerDataJSON{}, fmt.Errorf("prod build of %s needs a clean working tree: commit or stash your changes", g.ExactTag)
		}
	default:
		return LinkerDataJSON{}, fmt.Errorf("mode must be dev or prod, not %q", o.Mode)
	}
	d := LinkerDataJSON{
		Build:    o.Mode,
		Tag:      g.Version(),
		Hash:     g.Hash,
		Date:     g.CommitDate,
		AppName:  o.Name,
		AppTitle: strings.ReplaceAll(o.Title, " ", "_"),
	}
	if o.Mode == "dev" && !strings.Contains(d.Tag, "+") {
		d.Suffix = o.Branch // on a tag but a dev build: say which branch
	}
	return d, nil
}

// JSON is the linker data as the compact JSON SemVerJSON expects.
func (d LinkerDataJSON) JSON() string {
	b, _ := json.Marshal(d)
	return string(b)
}

// Ldflags is the -ldflags value that stamps d into a binary built with this
// package: go build -ldflags "$(clog BC genBuildinfo --format ldflags)".
func (d LinkerDataJSON) Ldflags() string {
	return "-X " + LinkerPath() + "='" + d.JSON() + "'"
}

// --- reading the repository ------------------------------------------------

const gitTimeout = 30 * time.Second

// git runs a read-only git command in dir and returns trimmed stdout.
func git(dir string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), gitTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok && len(ee.Stderr) > 0 {
			return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), strings.TrimSpace(string(ee.Stderr)))
		}
		return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return strings.TrimSpace(string(out)), nil
}

// releaseTags lists release tags newest first by tag date (annotated: tagger
// date, lightweight: commit date). extra narrows the list, e.g.
// "--merged", "origin/main" or "--points-at", "HEAD".
func releaseTags(dir string, extra ...string) ([]string, error) {
	args := append([]string{"for-each-ref", "--sort=-creatordate", "--format=%(refname:short)"}, extra...)
	args = append(args, "refs/tags")
	out, err := git(dir, args...)
	if err != nil {
		return nil, err
	}
	var tags []string
	for _, t := range strings.Split(out, "\n") {
		if IsReleaseTag(t) {
			tags = append(tags, t)
		}
	}
	return tags, nil
}

// LatestReleaseTag is the newest release tag by date - the production release.
// With branch set, only tags reachable from that branch count (e.g. origin/main
// for a scheduled rebuild).
func LatestReleaseTag(dir, branch string) (string, error) {
	var extra []string
	if branch != "" {
		if strings.HasPrefix(branch, "-") {
			return "", fmt.Errorf("unsafe branch %q", branch)
		}
		extra = []string{"--merged", branch}
	}
	tags, err := releaseTags(dir, extra...)
	if err != nil {
		return "", err
	}
	if len(tags) == 0 {
		where := "the repository"
		if branch != "" {
			where = branch
		}
		return "", fmt.Errorf("no release tag (vX.Y.Z) in %s", where)
	}
	return tags[0], nil
}

// ReadGitState reads the working copy in dir.
func ReadGitState(dir string) (GitState, error) {
	var g GitState
	var err error
	if g.Hash, err = git(dir, "rev-parse", "HEAD"); err != nil {
		return g, err
	}
	if g.CommitDate, err = git(dir, "log", "-1", "--format=%cs"); err != nil {
		return g, err
	}
	status, err := git(dir, "status", "--porcelain")
	if err != nil {
		return g, err
	}
	g.Dirty = status != ""

	if at, err := releaseTags(dir, "--points-at", "HEAD"); err == nil && len(at) > 0 {
		g.ExactTag = at[0]
		g.Nearest = at[0]
		return g, nil
	}
	reachable, err := releaseTags(dir, "--merged", "HEAD")
	if err != nil {
		return g, err
	}
	rng := "HEAD"
	if len(reachable) > 0 {
		g.Nearest = reachable[0]
		rng = g.Nearest + "..HEAD"
	}
	count, err := git(dir, "rev-list", "--count", rng)
	if err != nil {
		return g, err
	}
	g.Ahead, _ = strconv.Atoi(count)
	return g, nil
}
