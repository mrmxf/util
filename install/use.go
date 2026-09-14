//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/

package install

import (
	"fmt"
	"log/slog"
	"strings"
)

// useFlag and atFlag hold --use and its alias --at: an explicit version,
// "latest" or "lts". An empty value means "use the recipe's version spec".
var useFlag, atFlag string

const (
	useLatest = "latest"
	useLTS    = "lts"
)

// requestedVersion merges --use and --at, rejecting conflicting values.
func requestedVersion() (string, error) {
	if useFlag != "" && atFlag != "" && useFlag != atFlag {
		return "", fmt.Errorf("--use %q and --at %q disagree; give one version", useFlag, atFlag)
	}
	if useFlag != "" {
		return useFlag, nil
	}
	return atFlag, nil
}

// packageManagerStrategies install whatever version the package manager or
// vendor script provides, so they cannot honour a pinned version.
var packageManagerStrategies = map[string]bool{
	"brew-install": true,
	"apt-install":  true,
	"dnf-install":  true,
	"lnx-native":   true,
	"run-script":   true,
}

// applyUse resolves a --use request against the effective version and install
// specs. It returns the version to install and the install spec to use, which
// for go-install recipes is a copy with the import re-pinned to that version.
func applyUse(req string, verSpec *VersionSpec, inst *InstallSpec) (string, *InstallSpec, error) {
	req = strings.TrimSpace(req)
	slog.Info("version requested", "use", req, "source", "command line")

	if inst != nil && packageManagerStrategies[inst.Strategy] {
		if strings.EqualFold(req, useLatest) {
			logVersionMatch(useLatest, "package manager ("+inst.Strategy+")")
			return useLatest, inst, nil
		}
		return "", nil, fmt.Errorf("--use %s: %s installs the version the package manager provides; only --use latest is supported on this platform", req, inst.Strategy)
	}

	if inst != nil && inst.Strategy == "go-install" {
		return applyUseGoInstall(req, verSpec, inst)
	}

	switch strings.ToLower(req) {
	case useLatest:
		src := githubSource(verSpec)
		if src == nil {
			return "", nil, fmt.Errorf("--use latest: recipe has no GitHub version source to find the latest release")
		}
		v, err := resolveOnce(src)
		return v, inst, err
	case useLTS:
		src := githubSource(verSpec)
		if src == nil {
			return "", nil, fmt.Errorf("--use lts: recipe has no GitHub version source to find the lts release")
		}
		v, err := resolveLTS(src)
		return v, inst, err
	default:
		v, err := normaliseVersion(req, versionPrefix(verSpec))
		if err != nil {
			return "", nil, err
		}
		logVersionMatch(v, "command line --use", "requested", req)
		return v, inst, nil
	}
}

// applyUseGoInstall re-pins a go-install import path (module@version).
func applyUseGoInstall(req string, verSpec *VersionSpec, inst *InstallSpec) (string, *InstallSpec, error) {
	base := inst.Import
	if i := strings.LastIndex(base, "@"); i >= 0 {
		base = base[:i]
	}

	var ver string
	switch strings.ToLower(req) {
	case useLatest:
		ver = useLatest
	case useLTS:
		src := githubSource(verSpec)
		if src == nil {
			return "", nil, fmt.Errorf("--use lts: go-install recipe has no GitHub version source to find the lts release")
		}
		v, err := resolveLTS(src)
		if err != nil {
			return "", nil, err
		}
		ver = "v" + strings.TrimPrefix(v, "v")
	default:
		v, err := normaliseVersion(req, "")
		if err != nil {
			return "", nil, err
		}
		ver = "v" + v // Go module versions always carry a v
	}

	pinned := *inst
	pinned.Import = base + "@" + ver
	logVersionMatch(ver, "command line --use", "import", pinned.Import)
	return ver, &pinned, nil
}

// githubSource returns the first GitHub-based spec in the version chain
// (primary, then fallbacks), detached from its own fallback, or nil if none.
func githubSource(spec *VersionSpec) *VersionSpec {
	for s := spec; s != nil; s = s.Fallback {
		if s.Strategy == "github-latest" || s.Strategy == "github-tags" {
			src := *s
			src.Fallback = nil
			return &src
		}
	}
	return nil
}

// versionPrefix returns the first version-prefix declared in the version chain.
func versionPrefix(spec *VersionSpec) string {
	for s := spec; s != nil; s = s.Fallback {
		if s.VersionPrefix != "" {
			return s.VersionPrefix
		}
	}
	return ""
}

// normaliseVersion turns a user-supplied version (1.2.3, v1.2.3, go1.2.3) into
// the recipe's {version} form: prefix + bare version.
func normaliseVersion(req, prefix string) (string, error) {
	bare := req
	if prefix != "" {
		bare = strings.TrimPrefix(bare, prefix)
	}
	if len(bare) > 1 && bare[0] == 'v' {
		bare = bare[1:]
	}
	if bare == "" || bare[0] < '0' || bare[0] > '9' {
		return "", fmt.Errorf("--use %q: expected a version number, %s or %s", req, useLatest, useLTS)
	}
	return prefix + bare, nil
}

// resolveLTS returns the newest patch of the previous minor release line, e.g.
// 1.26.x when 1.27.x is current. None of the recipe tools publish a formal LTS,
// so this follows Go's support policy: the older of the two supported lines.
func resolveLTS(spec *VersionSpec) (string, error) {
	tags, err := fetchStableTags(spec)
	if err != nil {
		return "", err
	}
	lts, current, ok := pickLTS(tags)
	if !ok {
		return "", fmt.Errorf("lts: %s needs at least two stable minor release lines", spec.Repo)
	}
	ver := strings.TrimPrefix(lts.name, spec.TagPrefix)
	logVersionMatch(ver, "GitHub tags (lts: newest patch of previous minor line)",
		"repo", spec.Repo, "tag", lts.name, "current", current.name)
	return ver, nil
}

// pickLTS returns the newest tag of the second-highest major.minor line and
// the newest tag overall.
func pickLTS(tags []versionTag) (lts, current versionTag, ok bool) {
	if len(tags) == 0 {
		return versionTag{}, versionTag{}, false
	}
	current = tags[0]
	for _, t := range tags[1:] {
		if t.newer(current) {
			current = t
		}
	}
	found := false
	for _, t := range tags {
		sameLine := t.major == current.major && t.minor == current.minor
		if sameLine || t.newer(current) {
			continue
		}
		if !found || t.newer(lts) {
			lts, found = t, true
		}
	}
	return lts, current, found
}
