//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/

package ci

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// GitHub Release target data, per mode:
//
//	release:
//	  kind: github-release
//	  modes: [prod]
//	  prod:
//	    repo:   mrmxf/clog          owner/name (default: this repo's origin)
//	    tag:    "{tag}"             the release tag (required)
//	    dir:    _clog_build         where the assets are (required)
//	    assets: clog-amd-lnx …      space separated, relative to dir (required)
//	    notes:  path/to/notes.md    optional; else --generate-notes
//
// The checksums file is written from inside dir so its lines carry bare asset
// names. That is the form the installer greps for, and it refuses to install a
// binary it cannot verify - so assets and checksums are published together or
// not at all.
const releaseChecksums = "checksums.txt"

func deployGitHubRelease(d Deployment) error {
	tag, err := d.Get("tag", true)
	if err != nil {
		return err
	}
	dir, err := d.Get("dir", true)
	if err != nil {
		return err
	}
	assetList, err := d.Get("assets", true)
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
	if repo == "" {
		return fmt.Errorf("cannot work out which repo to publish to: set %s.%s.repo or add an origin remote",
			d.Target, d.Mode)
	}
	notes, err := d.Get("notes", false)
	if err != nil {
		return err
	}

	assets := strings.Fields(assetList)
	if len(assets) == 0 {
		return fmt.Errorf("ci.targets.%s.%s.assets is empty", d.Target, d.Mode)
	}
	for _, a := range assets {
		p := filepath.Join(dir, a)
		st, err := os.Stat(p)
		if err != nil {
			return fmt.Errorf("missing asset %s - run `clog build` first: %w", p, err)
		}
		if st.IsDir() {
			return fmt.Errorf("asset %s is a directory", p)
		}
	}

	// A release names a commit, so the tag has to exist before gh is asked to
	// point at one. Without this check `gh release create` invents the tag at
	// the default branch's head, which is silently the wrong commit whenever
	// the release is cut from anywhere else.
	if _, err := execCommand(".", "git", "rev-parse", "-q", "--verify", "refs/tags/"+tag); err != nil {
		return fmt.Errorf("no local tag %s - tag the release commit first", tag)
	}
	if _, err := execCommand(".", "git", "ls-remote", "--exit-code", "--tags", "origin", "refs/tags/"+tag); err != nil {
		return fmt.Errorf("tag %s is not at origin - git push origin %s", tag, tag)
	}

	slog.Info("github-release", "repo", repo, "tag", tag, "dir", dir, "assets", len(assets))

	if d.DryRun {
		fmt.Fprintf(d.Out, "dry-run: would publish %s to %s with %d assets + %s\n",
			tag, repo, len(assets), releaseChecksums)
		return nil
	}

	sumPath := filepath.Join(dir, releaseChecksums)
	if err := writeChecksums(dir, assets, sumPath); err != nil {
		return err
	}
	fmt.Fprintf(d.Out, "%s over %d assets\n", releaseChecksums, len(assets))

	files := make([]string, 0, len(assets)+1)
	for _, a := range assets {
		files = append(files, filepath.Join(dir, a))
	}
	files = append(files, sumPath)

	// Re-running a release replaces its assets rather than failing, so a re-run
	// after a half-finished publish is safe. Deleting and re-cutting the tag
	// would not be: the installer verifies the checksum, and anyone who already
	// downloaded would see a mismatch rather than an update.
	if _, err := execCommand(".", "gh", "release", "view", tag, "--repo", repo); err == nil {
		slog.Info("release exists - replacing assets", "tag", tag)
		args := append([]string{"release", "upload", tag}, files...)
		args = append(args, "--repo", repo, "--clobber")
		if out, err := execCommand(".", "gh", args...); err != nil {
			return fmt.Errorf("gh release upload: %w%s", err, indentOutput(out))
		}
	} else {
		args := append([]string{"release", "create", tag}, files...)
		args = append(args, "--repo", repo, "--title", tag)
		if notes != "" {
			args = append(args, "--notes-file", notes)
		} else {
			args = append(args, "--generate-notes")
		}
		if out, err := execCommand(".", "gh", args...); err != nil {
			return fmt.Errorf("gh release create: %w%s", err, indentOutput(out))
		}
	}

	fmt.Fprintf(d.Out, "published %s to github.com/%s/releases/tag/%s\n", tag, repo, tag)
	return nil
}

// writeChecksums writes `<sha256>  <name>` lines for assets, with bare names,
// sorted so the file is reproducible whatever order the config listed them in.
func writeChecksums(dir string, assets []string, out string) error {
	names := append([]string(nil), assets...)
	sort.Strings(names)

	var b strings.Builder
	for _, name := range names {
		sum, err := sha256File(filepath.Join(dir, name))
		if err != nil {
			return err
		}
		// two spaces: the format sha256sum reads back with -c
		fmt.Fprintf(&b, "%s  %s\n", sum, name)
	}
	if err := os.WriteFile(out, []byte(b.String()), 0o644); err != nil {
		return fmt.Errorf("cannot write %s: %w", out, err)
	}
	return nil
}

func sha256File(path string) (string, error) {
	f, err := os.Open(filepath.Clean(path))
	if err != nil {
		return "", fmt.Errorf("cannot read %s: %w", path, err)
	}
	defer f.Close() //nolint:errcheck // read-only

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("cannot hash %s: %w", path, err)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
