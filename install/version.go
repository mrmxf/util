//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/

package install

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// ResolveVersion dispatches to the appropriate resolver for spec.Strategy.
// On failure, if spec.Fallback is non-nil it is tried before giving up.
func ResolveVersion(spec *VersionSpec) (string, error) {
	if spec == nil {
		return "", fmt.Errorf("version spec is nil")
	}
	ver, err := resolveOnce(spec)
	if err != nil && spec.Fallback != nil {
		slog.Warn("version resolve failed, trying fallback", "strategy", spec.Strategy, "err", err)
		return resolveOnce(spec.Fallback)
	}
	return ver, err
}

func resolveOnce(spec *VersionSpec) (string, error) {
	switch spec.Strategy {
	case "go-mod":
		return resolveGoMod(spec)
	case "hugo-module":
		return resolveHugoModule(spec)
	case "github-latest":
		return resolveGitHubLatest(spec)
	case "github-tags":
		return resolveGitHubTags(spec)
	case "pinned":
		return resolvePinned(spec)
	default:
		return "", fmt.Errorf("unknown version strategy %q", spec.Strategy)
	}
}

// resolveGoMod reads the Go version from go.mod (e.g. "go 1.22.3" → "go1.22.3").
var goVerRE = regexp.MustCompile(`\d+\.\d+\.\d+`)

func resolveGoMod(spec *VersionSpec) (string, error) {
	file := spec.File
	if file == "" {
		file = "go.mod"
	}
	f, err := os.Open(file)
	if err != nil {
		return "", fmt.Errorf("go-mod: cannot open %q: %w", file, err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "go ") {
			field := strings.TrimSpace(strings.TrimPrefix(line, "go "))
			// Prefer X.Y.Z; accept X.Y and pad with .0
			if goVerRE.MatchString(field) {
				return "go" + goVerRE.FindString(field), nil
			}
			// X.Y only — pad to X.Y.0
			parts := strings.Split(field, ".")
			if len(parts) == 2 {
				return "go" + field + ".0", nil
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("go-mod: read error in %q: %w", file, err)
	}
	return "", fmt.Errorf("go-mod: no 'go X.Y' directive found in %q", file)
}

// resolveHugoModule reads hugoVersion.min from a Hugo module config file.
func resolveHugoModule(spec *VersionSpec) (string, error) {
	file := spec.File
	if file == "" {
		file = "config/_default/module.yaml"
	}
	data, err := os.ReadFile(file)
	if err != nil {
		return "", fmt.Errorf("hugo-module: cannot read %q: %w", file, err)
	}
	var m struct {
		HugoVersion struct {
			Min string `yaml:"min"`
		} `yaml:"hugoVersion"`
	}
	if err := yaml.Unmarshal(data, &m); err != nil {
		return "", fmt.Errorf("hugo-module: cannot parse %q: %w", file, err)
	}
	if m.HugoVersion.Min == "" {
		return "", fmt.Errorf("hugo-module: hugoVersion.min not found in %q", file)
	}
	return strings.TrimPrefix(m.HugoVersion.Min, "v"), nil
}

// resolveGitHubLatest fetches the latest release tag from the GitHub releases API.
// If the releases/latest endpoint returns 404 (repo uses tags rather than GitHub
// releases), it automatically retries via resolveGitHubTags with the same Repo
// and TagPrefix so callers don't need a manual fallback for this common case.
func resolveGitHubLatest(spec *VersionSpec) (string, error) {
	if spec.Repo == "" {
		return "", fmt.Errorf("github-latest: repo field is required")
	}
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", spec.Repo)
	resp, err := http.Get(url) //nolint:gosec // URL is built from trusted config
	if err != nil {
		return "", fmt.Errorf("github-latest: GET %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		slog.Warn("github-latest: releases/latest returned 404, falling back to github-tags", "repo", spec.Repo)
		return resolveGitHubTags(spec)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("github-latest: %s returned HTTP %d", url, resp.StatusCode)
	}
	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", fmt.Errorf("github-latest: decode %s: %w", url, err)
	}
	ver := release.TagName
	if spec.TagPrefix != "" {
		ver = strings.TrimPrefix(ver, spec.TagPrefix)
	}
	return ver, nil
}

// defaultTagExclude matches common pre-release suffixes: rc1, beta2, alpha, pre3.
var defaultTagExclude = regexp.MustCompile(`(?i)(rc|beta|alpha|pre)\d*$`)

// tagVerRE extracts a comparable version tuple from a tag name regardless of
// prefix (handles "v1.2.3", "go1.2.3", "release-1.2.3", etc.).
var tagVerRE = regexp.MustCompile(`(\d+)\.(\d+)(?:\.(\d+))?`)

// resolveGitHubTags paginates the GitHub tags API (up to 5 pages / 500 tags),
// filters candidates, then semver-sorts to return the highest stable version.
// This handles repos like golang/go whose tag order is not purely newest-first.
//
// Fields used from spec:
//   - Repo:       "owner/repo" (required)
//   - TagFilter:  regexp; only matching tags are candidates (empty = all)
//   - TagExclude: regexp; matching tags are skipped (empty = default pre-release filter)
//   - TagPrefix:  strip from the winning tag to form the version (empty = return full tag)
func resolveGitHubTags(spec *VersionSpec) (string, error) {
	if spec.Repo == "" {
		return "", fmt.Errorf("github-tags: repo field is required")
	}

	var tagFilter *regexp.Regexp
	if spec.TagFilter != "" {
		var err error
		tagFilter, err = regexp.Compile(spec.TagFilter)
		if err != nil {
			return "", fmt.Errorf("github-tags: invalid tag-filter %q: %w", spec.TagFilter, err)
		}
	}
	tagExclude := defaultTagExclude
	if spec.TagExclude != "" {
		var err error
		tagExclude, err = regexp.Compile(spec.TagExclude)
		if err != nil {
			return "", fmt.Errorf("github-tags: invalid tag-exclude %q: %w", spec.TagExclude, err)
		}
	}

	type vtag struct {
		name                string
		major, minor, patch int
	}
	var best *vtag

	const maxPages = 5
	for page := 1; page <= maxPages; page++ {
		url := fmt.Sprintf("https://api.github.com/repos/%s/tags?per_page=100&page=%d", spec.Repo, page)
		resp, err := http.Get(url) //nolint:gosec // URL is built from trusted config
		if err != nil {
			return "", fmt.Errorf("github-tags: GET %s: %w", url, err)
		}
		var tags []struct {
			Name string `json:"name"`
		}
		decodeErr := json.NewDecoder(resp.Body).Decode(&tags)
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("github-tags: %s returned HTTP %d", url, resp.StatusCode)
		}
		if decodeErr != nil {
			return "", fmt.Errorf("github-tags: decode %s: %w", url, decodeErr)
		}

		for _, tag := range tags {
			name := tag.Name
			if tagFilter != nil && !tagFilter.MatchString(name) {
				continue
			}
			if tagExclude.MatchString(name) {
				continue
			}
			m := tagVerRE.FindStringSubmatch(name)
			if m == nil {
				continue // no parseable version numbers
			}
			maj, _ := strconv.Atoi(m[1])
			min, _ := strconv.Atoi(m[2])
			pat := 0
			if m[3] != "" {
				pat, _ = strconv.Atoi(m[3])
			}
			if best == nil ||
				maj > best.major ||
				(maj == best.major && min > best.minor) ||
				(maj == best.major && min == best.minor && pat > best.patch) {
				best = &vtag{name: name, major: maj, minor: min, patch: pat}
			}
		}

		if len(tags) < 100 {
			break // last page
		}
	}

	if best == nil {
		return "", fmt.Errorf("github-tags: no stable tag found in %s (filter=%q, exclude=%q)",
			spec.Repo, spec.TagFilter, spec.TagExclude)
	}
	ver := best.name
	if spec.TagPrefix != "" {
		ver = strings.TrimPrefix(best.name, spec.TagPrefix)
	}
	return ver, nil
}

// resolvePinned returns the fixed version from spec.Value.
func resolvePinned(spec *VersionSpec) (string, error) {
	if spec.Value == "" {
		return "", fmt.Errorf("pinned: value field is required")
	}
	return spec.Value, nil
}
