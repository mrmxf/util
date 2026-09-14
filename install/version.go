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
	"path/filepath"
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
	if Verbose {
		attrs := []any{"strategy", spec.Strategy}
		for _, kv := range [][2]string{{"file", spec.File}, {"repo", spec.Repo}, {"tag-filter", spec.TagFilter}, {"value", spec.Value}} {
			if kv[1] != "" {
				attrs = append(attrs, kv[0], kv[1])
			}
		}
		vlog("version lookup", attrs...)
	}
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

// logVersionMatch reports, at INFO, the version found and where it came from.
func logVersionMatch(version, source string, attrs ...any) {
	slog.Info("version match", append([]any{"version", version, "source", source}, attrs...)...)
}

// absPath returns file as an absolute path for logging, or file unchanged.
func absPath(file string) string {
	if abs, err := filepath.Abs(file); err == nil {
		return abs
	}
	return file
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
				v := "go" + goVerRE.FindString(field)
				logVersionMatch(v, "go.mod go directive", "path", absPath(file), "line", line)
				return v, nil
			}
			// X.Y only — pad to X.Y.0
			parts := strings.Split(field, ".")
			if len(parts) == 2 {
				v := "go" + field + ".0"
				logVersionMatch(v, "go.mod go directive (padded to X.Y.0)", "path", absPath(file), "line", line)
				return v, nil
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
	v := strings.TrimPrefix(m.HugoVersion.Min, "v")
	logVersionMatch(v, "hugo module config hugoVersion.min", "path", absPath(file), "min", m.HugoVersion.Min)
	return v, nil
}

// resolveGitHubLatest fetches the latest release tag from the GitHub releases API.
// If the releases/latest endpoint returns 404 (repo uses tags rather than GitHub
// releases), it automatically retries via resolveGitHubTags with the same Repo
// and TagPrefix so callers don't need a manual fallback for this common case.
func resolveGitHubLatest(spec *VersionSpec) (string, error) {
	if spec.Repo == "" {
		return "", fmt.Errorf("github-latest: repo field is required")
	}
	url := fmt.Sprintf("%s/repos/%s/releases/latest", githubAPI, spec.Repo)
	resp, err := githubGet(url)
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
	logVersionMatch(ver, "GitHub latest release", "repo", spec.Repo, "tag", release.TagName)
	return ver, nil
}

// defaultTagExclude matches common pre-release suffixes: rc1, beta2, alpha, pre3.
var defaultTagExclude = regexp.MustCompile(`(?i)(rc|beta|alpha|pre)\d*$`)

// tagVerRE extracts a comparable version tuple from a tag name regardless of
// prefix (handles "v1.2.3", "go1.2.3", "release-1.2.3", etc.).
var tagVerRE = regexp.MustCompile(`(\d+)\.(\d+)(?:\.(\d+))?`)

// versionTag is a tag name with its parsed major.minor.patch.
type versionTag struct {
	name                string
	major, minor, patch int
}

// newer reports whether t is a higher version than o.
func (t versionTag) newer(o versionTag) bool {
	if t.major != o.major {
		return t.major > o.major
	}
	if t.minor != o.minor {
		return t.minor > o.minor
	}
	return t.patch > o.patch
}

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
	tags, err := fetchStableTags(spec)
	if err != nil {
		return "", err
	}
	var best *versionTag
	for i := range tags {
		if best == nil || tags[i].newer(*best) {
			best = &tags[i]
		}
	}
	if best == nil {
		return "", fmt.Errorf("github-tags: no stable tag found in %s (filter=%q, exclude=%q)",
			spec.Repo, spec.TagFilter, spec.TagExclude)
	}
	ver := strings.TrimPrefix(best.name, spec.TagPrefix)
	logVersionMatch(ver, "GitHub tags (highest stable)", "repo", spec.Repo, "tag", best.name, "tag-filter", spec.TagFilter)
	return ver, nil
}

// fetchStableTags returns every tag in spec.Repo that passes the tag filter and
// pre-release exclusion and has a parseable version number.
func fetchStableTags(spec *VersionSpec) ([]versionTag, error) {
	if spec.Repo == "" {
		return nil, fmt.Errorf("github-tags: repo field is required")
	}

	var tagFilter *regexp.Regexp
	if spec.TagFilter != "" {
		var err error
		tagFilter, err = regexp.Compile(spec.TagFilter)
		if err != nil {
			return nil, fmt.Errorf("github-tags: invalid tag-filter %q: %w", spec.TagFilter, err)
		}
	}
	tagExclude := defaultTagExclude
	if spec.TagExclude != "" {
		var err error
		tagExclude, err = regexp.Compile(spec.TagExclude)
		if err != nil {
			return nil, fmt.Errorf("github-tags: invalid tag-exclude %q: %w", spec.TagExclude, err)
		}
	}

	var out []versionTag
	const maxPages = 5
	for page := 1; page <= maxPages; page++ {
		url := fmt.Sprintf("%s/repos/%s/tags?per_page=100&page=%d", githubAPI, spec.Repo, page)
		resp, err := githubGet(url)
		if err != nil {
			return nil, fmt.Errorf("github-tags: GET %s: %w", url, err)
		}
		var tags []struct {
			Name string `json:"name"`
		}
		decodeErr := json.NewDecoder(resp.Body).Decode(&tags)
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("github-tags: %s returned HTTP %d", url, resp.StatusCode)
		}
		if decodeErr != nil {
			return nil, fmt.Errorf("github-tags: decode %s: %w", url, decodeErr)
		}

		for _, tag := range tags {
			if tagFilter != nil && !tagFilter.MatchString(tag.Name) {
				continue
			}
			if tagExclude.MatchString(tag.Name) {
				continue
			}
			if t, ok := parseVersionTag(tag.Name); ok {
				out = append(out, t)
			}
		}

		if len(tags) < 100 {
			break // last page
		}
	}
	return out, nil
}

// parseVersionTag extracts major.minor[.patch] from a tag name.
func parseVersionTag(name string) (versionTag, bool) {
	m := tagVerRE.FindStringSubmatch(name)
	if m == nil {
		return versionTag{}, false
	}
	t := versionTag{name: name}
	t.major, _ = strconv.Atoi(m[1])
	t.minor, _ = strconv.Atoi(m[2])
	if m[3] != "" {
		t.patch, _ = strconv.Atoi(m[3])
	}
	return t, true
}

// resolvePinned returns the fixed version from spec.Value.
func resolvePinned(spec *VersionSpec) (string, error) {
	if spec.Value == "" {
		return "", fmt.Errorf("pinned: value field is required")
	}
	logVersionMatch(spec.Value, "pinned in recipe")
	return spec.Value, nil
}
