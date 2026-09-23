//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/

package ci

import (
	"fmt"
	"strconv"
	"strings"
)

// Scanning has two axes, because vulnerabilities and secrets are independent
// questions about the same subject.
//
//   - source sweeps the worktree. It runs in every repo, with or without
//     targets, which is what covers a Go library, a Hugo asset pipeline and a
//     TinyGo module graph. Three of the four repo shapes have no scannable
//     deployable at all.
//   - artifact sweeps what a target ships, once per target, after the build.
//
// The value names what is examined, never which tool does it: swapping Trivy is
// then a change in one check block rather than a config migration across every
// repo. See ci/SCANNING.md.

// Vulnerability sweep subjects.
const (
	VulnImage = "image" // a container image or an image tarball
	VulnFS    = "fs"    // a directory of files, including compiled binaries
	VulnRepo  = "repo"  // the whole worktree, lockfiles and all
	VulnNone  = "none"  // nothing to examine - and you have to write it down
)

var knownVulnSweeps = []string{VulnImage, VulnFS, VulnRepo, VulnNone}

// ScanKey is the .clog.yaml key holding scan configuration.
const ScanKey = ConfigKey + ".scan"

// ScanConfig is ci.scan.
type ScanConfig struct {
	Source ScanAxis `json:"source"`
	// Artifact holds the repo-wide defaults a target's kind and its own scan
	// block then refine.
	Artifact ScanAxis `json:"artifact"`
}

// ScanAxis is one sweep: what to look at, and how hard to be about it. Secrets
// is a *string so an unset value can take the default while an explicit false
// survives.
type ScanAxis struct {
	Vuln          string `json:"vuln"`
	Secrets       *bool  `json:"secrets"`
	Ref           string `json:"ref"`
	Severity      string `json:"severity"`
	IgnoreUnfixed *bool  `json:"ignore-unfixed"`
	IgnoreFile    string `json:"ignorefile"`
	SkipDirs      string `json:"skip-dirs"`
}

// Defaults. ignore-unfixed is on because base images accumulate HIGH and
// CRITICAL findings with no fix available, and a blocking gate without it
// produces chronic red - which gets scanning switched off altogether.
const (
	defaultSourceVuln      = VulnRepo
	defaultSourceRef       = "."
	defaultSourceSeverity  = "HIGH,CRITICAL"
	defaultSourceSkipDirs  = "node_modules,vendor,public,tmp"
	defaultArtifactSevrity = "CRITICAL"
	defaultIgnoreFile      = ".trivyignore"
)

// kindScan is a target kind's artifact-sweep default. A zero Vuln means the
// kind has no honest default and the target must say so itself.
type kindScan struct {
	Vuln     string
	Required bool // no default exists: `vuln:` must be written down
	RefKey   string
}

// kindScanDefaults. The Pages and bucket kinds deliberately have no default: a
// filesystem vulnerability sweep of a rendered site finds nothing and reports
// it in green, which is worse than not scanning, and silently defaulting to
// `none` is the exact failure this design exists to remove.
var kindScanDefaults = map[string]kindScan{
	KindRegistry:       {Vuln: VulnImage, RefKey: "image"},
	KindGitHubRelease:  {Vuln: VulnFS, RefKey: "dir"},
	KindGitLabRelease:  {Vuln: VulnFS, RefKey: "dir"},
	KindGitHubPages:    {Required: true, RefKey: "dir"},
	KindGitLabPages:    {Required: true, RefKey: "dir"},
	KindCloudflarePage: {Required: true, RefKey: "dir"},
	KindBucket:         {Required: true, RefKey: "dir"},
}

func boolOr(p *bool, def bool) bool {
	if p == nil {
		return def
	}
	return *p
}

func strOr(s, def string) string {
	if strings.TrimSpace(s) == "" {
		return def
	}
	return s
}

// validateVuln rejects a sweep name that is not one of the four.
func validateVuln(where, v string) error {
	for _, k := range knownVulnSweeps {
		if v == k {
			return nil
		}
	}
	return fmt.Errorf("%s vuln %q is not one of %s", where, v, strings.Join(knownVulnSweeps, ", "))
}

// SourceScan resolves the repo-wide sweep. It never errors on a missing config,
// because the honest default for a worktree is to look at all of it.
func SourceScan(cfg Config) (ScanAxis, error) {
	a := cfg.Scan.Source
	a.Vuln = strOr(a.Vuln, defaultSourceVuln)
	if err := validateVuln(ScanKey+".source", a.Vuln); err != nil {
		return ScanAxis{}, err
	}
	a.Ref = strOr(a.Ref, defaultSourceRef)
	a.Severity = strOr(a.Severity, defaultSourceSeverity)
	a.SkipDirs = strOr(a.SkipDirs, defaultSourceSkipDirs)
	a.IgnoreFile = strOr(a.IgnoreFile, defaultIgnoreFile)
	yes, no := true, true
	if a.Secrets == nil {
		a.Secrets = &yes
	}
	if a.IgnoreUnfixed == nil {
		a.IgnoreUnfixed = &no
	}
	return a, nil
}

// ArtifactScan resolves the sweep for one target, most specific winning:
// the target's own scan block, then ci.scan.artifact, then the kind's default.
//
// A kind with no honest default is an error naming the target and saying what
// to write. One line, written once, recording a decision somebody made - as
// against silence that means "not scanned", which is the thing this whole
// design exists to remove.
func ArtifactScan(env Env, cfg Config, mode, name string) (ScanAxis, error) {
	t, ok := cfg.Targets[name]
	if !ok {
		return ScanAxis{}, fmt.Errorf("no ci.targets.%s in the clog config (have: %s)", name, strings.Join(sortedKeys(cfg.Targets), ", "))
	}
	if err := t.validate(name); err != nil {
		return ScanAxis{}, err
	}
	kd, known := kindScanDefaults[t.Kind]
	if !known {
		return ScanAxis{}, fmt.Errorf("ci.targets.%s kind %q has no scan behaviour defined", name, t.Kind)
	}

	a := t.Scan // most specific
	if strings.TrimSpace(a.Vuln) == "" {
		a.Vuln = cfg.Scan.Artifact.Vuln
	}
	if strings.TrimSpace(a.Vuln) == "" {
		if kd.Required {
			return ScanAxis{}, fmt.Errorf(
				"ci.targets.%s is a %s target, so there is no honest default sweep for it: write `scan: {vuln: none}` to record that its output has no dependencies to scan, or name a sweep (%s)",
				name, t.Kind, strings.Join(knownVulnSweeps, ", "))
		}
		a.Vuln = kd.Vuln
	}
	if err := validateVuln("ci.targets."+name+".scan", a.Vuln); err != nil {
		return ScanAxis{}, err
	}

	if a.Secrets == nil {
		a.Secrets = cfg.Scan.Artifact.Secrets
	}
	if a.Secrets == nil {
		yes := true
		a.Secrets = &yes // an honest default everywhere, so never required
	}
	if a.IgnoreUnfixed == nil {
		a.IgnoreUnfixed = cfg.Scan.Artifact.IgnoreUnfixed
	}
	if a.IgnoreUnfixed == nil {
		yes := true
		a.IgnoreUnfixed = &yes
	}
	a.Severity = strOr(a.Severity, strOr(cfg.Scan.Artifact.Severity, defaultArtifactSevrity))
	a.IgnoreFile = strOr(a.IgnoreFile, strOr(cfg.Scan.Artifact.IgnoreFile, defaultIgnoreFile))
	a.SkipDirs = strOr(a.SkipDirs, cfg.Scan.Artifact.SkipDirs)

	// The reference is per-mode data, not a field on Target: `dir` and `image`
	// live inside the dev:/prod: maps. Resolve it the same way `ci target get`
	// does, so a dev build asking for a prod-only dir fails by name.
	if strings.TrimSpace(a.Ref) == "" && a.Vuln != VulnNone {
		ref, err := TargetGet(env, cfg, mode, name, kd.RefKey, false)
		if err != nil {
			return ScanAxis{}, err
		}
		if strings.TrimSpace(ref) == "" {
			return ScanAxis{}, fmt.Errorf(
				"ci.targets.%s.%s.%s is not set, so the %s sweep has nothing to point at (set it, or scan: {ref: …})",
				name, mode, kd.RefKey, a.Vuln)
		}
		a.Ref = ref
	}
	return a, nil
}

// ScanTargets lists the targets an artifact sweep should visit.
//
// It deliberately does NOT filter by deploy-mode membership. `clog CI target list`
// does, and a target with only a prod: block therefore reports no dev
// membership - so enumerating with it would leave prod-only targets unscanned
// on every pull request, which is the original bug through a different door.
func ScanTargets(cfg Config, selected, all []Stack) ([]string, error) {
	names, err := TargetsForStacks(cfg, selected, all)
	if err != nil {
		return nil, err
	}
	for _, n := range names {
		if err := cfg.Targets[n].validate(n); err != nil {
			return nil, err
		}
	}
	return names, nil
}

// EnvLines renders one axis as shell KEY=value lines for $GITHUB_ENV.
func (a ScanAxis) EnvLines(prefix string) string {
	return "" +
		prefix + "_vuln=" + a.Vuln + "\n" +
		prefix + "_secrets=" + strconv.FormatBool(boolOr(a.Secrets, true)) + "\n" +
		prefix + "_ref=" + a.Ref + "\n" +
		prefix + "_severity=" + a.Severity + "\n" +
		prefix + "_ignore_unfixed=" + strconv.FormatBool(boolOr(a.IgnoreUnfixed, true)) + "\n" +
		prefix + "_ignorefile=" + a.IgnoreFile + "\n" +
		prefix + "_skip_dirs=" + a.SkipDirs + "\n"
}
