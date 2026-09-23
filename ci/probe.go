//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/

package ci

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// A probe examines what is LIVE. A scan examines what sits still.
//
// That is the whole distinction, and it is why the two are different words
// with different directories. Scanning reads source, lockfiles and binaries,
// so it runs at build time on every branch and pull request - it needs nothing
// deployed, and gating it on deployment is the bug that left pull requests
// unswept. A probe cannot run before a deploy, because until then there is
// nothing to ask: a bucket that is not yet public cannot be publicly listable,
// and a site that is not yet served cannot be serving its own .git directory.
//
// Probes report rather than block by default. A deploy has already happened by
// the time they run, so failing the job cannot un-publish anything; what it
// can do is tell you, loudly, in the same run. `--strict` turns findings into
// a non-zero exit for pipelines that gate a promotion on them.

// ProbeLevel is how much a finding matters.
type ProbeLevel string

const (
	ProbePass ProbeLevel = "pass"
	ProbeWarn ProbeLevel = "warn"
	ProbeFail ProbeLevel = "fail"
)

// ProbeFinding is one question asked of a live target, and its answer.
type ProbeFinding struct {
	Target string     `json:"target"`
	Kind   string     `json:"kind"`
	Check  string     `json:"check"`
	Level  ProbeLevel `json:"level"`
	Detail string     `json:"detail"`
}

// ProbeReport is everything asked of one run's targets.
type ProbeReport struct {
	Mode     string         `json:"mode"`
	When     string         `json:"when"`
	Findings []ProbeFinding `json:"findings"`
}

// Failed reports whether anything came back at ProbeFail.
func (r ProbeReport) Failed() bool {
	for _, f := range r.Findings {
		if f.Level == ProbeFail {
			return true
		}
	}
	return false
}

// Prober asks one kind of target its questions.
type Prober func(Deployment) []ProbeFinding

// probers maps a target kind to its implementation. A kind with no prober is
// reported as such rather than skipped silently: "nothing was asked" and
// "nothing was wrong" must never look the same.
var probers = map[string]Prober{
	KindGitHubPages:    probeWebsite,
	KindGitLabPages:    probeWebsite,
	KindCloudflarePage: probeWebsite,
	KindBucket:         probeBucket,
	KindRegistry:       probeRegistry,
	KindGitHubRelease:  probeRelease,
	KindGitLabRelease:  probeRelease,
}

// httpGet is the seam every probe reaches the network through, so the whole
// set is testable without a live target.
var httpGet = func(url string) (status int, body string, err error) {
	c := &http.Client{Timeout: 15 * time.Second}
	resp, err := c.Get(url) //nolint:gosec // the url comes from validated config
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close() //nolint:errcheck // read-only
	// A probe only needs enough to recognise what it is looking at; a
	// misconfigured bucket can otherwise return a very large listing.
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	return resp.StatusCode, string(b), nil
}

func pass(d Deployment, check, detail string) ProbeFinding {
	return finding(d, check, ProbePass, detail)
}
func warn(d Deployment, check, detail string) ProbeFinding {
	return finding(d, check, ProbeWarn, detail)
}
func fail(d Deployment, check, detail string) ProbeFinding {
	return finding(d, check, ProbeFail, detail)
}
func finding(d Deployment, check string, lvl ProbeLevel, detail string) ProbeFinding {
	kind := ""
	if t, ok := d.Cfg.Targets[d.Target]; ok {
		kind = t.Kind
	}
	return ProbeFinding{Target: d.Target, Kind: kind, Check: check, Level: lvl, Detail: detail}
}

// Probe runs the prober for every target that deployed in mode, and writes the
// report to _clog_deploy/probe/. When only is set, just that target runs.
func Probe(env Env, cfg Config, mode, only string, dryRun bool, out io.Writer) (ProbeReport, error) {
	report := ProbeReport{Mode: mode, When: time.Now().UTC().Format(time.RFC3339)}

	names, err := TargetNames(cfg, mode, "")
	if err != nil {
		return report, err
	}
	if only != "" {
		if !contains(names, only) {
			return report, fmt.Errorf("ci.targets.%s does not deploy in %s mode", only, mode)
		}
		names = []string{only}
	}
	if len(names) == 0 {
		slog.Warn("no deploy targets for this mode, so nothing to probe", "mode", mode)
		return report, nil
	}

	for _, name := range names {
		t := cfg.Targets[name]
		d := Deployment{Env: env, Cfg: cfg, Mode: mode, Target: name, DryRun: dryRun, Out: out}
		run, ok := probers[t.Kind]
		if !ok {
			report.Findings = append(report.Findings,
				finding(d, "probe available", ProbeWarn,
					fmt.Sprintf("kind %q has no probe, so nothing was checked", t.Kind)))
			continue
		}
		if dryRun {
			report.Findings = append(report.Findings,
				finding(d, "probe available", ProbePass, "dry-run: would probe this target"))
			continue
		}
		slog.Info("probe target", "target", name, "kind", t.Kind, "mode", mode)
		report.Findings = append(report.Findings, run(d)...)
	}

	sort.SliceStable(report.Findings, func(i, j int) bool {
		return report.Findings[i].Target < report.Findings[j].Target
	})
	return report, nil
}

// WriteProbeReport saves the report under the canonical deploy artifact
// directory, so a debugger knows where to look instead of guessing.
func WriteProbeReport(r ProbeReport, dir string) (string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("cannot create %s: %w", dir, err)
	}
	path := filepath.Join(dir, "probe.json")
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, append(b, '\n'), 0o644); err != nil {
		return "", fmt.Errorf("cannot write %s: %w", path, err)
	}
	return path, nil
}

// FormatProbe writes a human summary, one line per finding.
func FormatProbe(r ProbeReport, out io.Writer) {
	for _, f := range r.Findings {
		mark := "ok  "
		switch f.Level {
		case ProbeWarn:
			mark = "warn"
		case ProbeFail:
			mark = "FAIL"
		}
		fmt.Fprintf(out, "%s %-12s %-28s %s\n", mark, f.Target, f.Check, f.Detail)
	}
}

// trimURL makes a base URL safe to join paths onto.
func trimURL(s string) string { return strings.TrimRight(s, "/") }
