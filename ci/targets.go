//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/

package ci

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// TargetVar names the target a deploy step is working on. The deploy runs once
// per target (D-I.14), each time with this set.
const TargetVar = "CLOG_TARGET"

// Target kinds. A deploy sends the build to exactly one kind of destination;
// a project with two destinations declares two targets.
const (
	KindRegistry       = "container-registry" // image push (+ optional webhook)
	KindBucket         = "bucket"             // object store (S3-compatible)
	KindCloudflarePage = "cloudflare-pages"
	KindGitHubPages    = "github-pages"
	KindGitLabPages    = "gitlab-pages"
	KindGitHubRelease  = "github-release" // release assets, published with gh
	KindGitLabRelease  = "gitlab-release" // release assets, published with glab
)

// knownKinds. `package` was removed: whether it meant a built tarball or a
// source tree was never settled, no repo used it, and a kind whose scan
// subject is unknowable is a kind that cannot have an honest default.
var knownKinds = []string{
	KindRegistry, KindBucket, KindCloudflarePage,
	KindGitHubPages, KindGitLabPages,
	KindGitHubRelease, KindGitLabRelease,
}

// Target is one deploy destination: ci.targets.<name>.
type Target struct {
	Kind    string         `json:"kind"`
	Require []string       `json:"require"` // secret names this destination needs
	Modes   []string       `json:"modes"`   // modes it deploys in (default: dev + prod)
	Dev     map[string]any `json:"dev"`     // per-mode data …
	Prod    map[string]any `json:"prod"`    // … read with `clog ci target get`
	Comment string         `json:"comment,omitempty"`
	// Stack binds the target to one ci.stack entry, so `clog deploy bonfire`
	// never publishes an artifact this run did not build. Empty means the first
	// stack, so a single-stack repo never learns the field exists.
	Stack string `json:"stack"`
	// Scan overrides the artifact sweep the kind would otherwise supply.
	Scan ScanAxis `json:"scan"`
}

// data returns the target's data for a mode.
func (t Target) data(mode string) map[string]any {
	if mode == ModeProd {
		return t.Prod
	}
	return t.Dev
}

// deploysIn reports whether the target is used in a mode.
func (t Target) deploysIn(mode string) bool {
	if len(t.Modes) == 0 {
		return t.data(mode) != nil
	}
	for _, m := range t.Modes {
		if strings.TrimSpace(m) == mode {
			return true
		}
	}
	return false
}

// validate checks the kind against the known list.
func (t Target) validate(name string) error {
	if t.Kind == "" {
		return fmt.Errorf("ci.targets.%s.kind is not set (one of %s)", name, strings.Join(knownKinds, ", "))
	}
	for _, k := range knownKinds {
		if t.Kind == k {
			return nil
		}
	}
	return fmt.Errorf("ci.targets.%s.kind %q is not one of %s", name, t.Kind, strings.Join(knownKinds, ", "))
}

// TargetNames lists the targets that deploy in a mode, optionally filtered by
// kind, in config order made stable by sorting.
func TargetNames(cfg Config, mode, kind string) ([]string, error) {
	var names []string
	for _, name := range sortedKeys(cfg.Targets) {
		t := cfg.Targets[name]
		if err := t.validate(name); err != nil {
			return nil, err
		}
		if !t.deploysIn(mode) || (kind != "" && t.Kind != kind) {
			continue
		}
		names = append(names, name)
	}
	return names, nil
}

// TargetGet returns one value from the current target's data for a mode, with
// {tag} {version} {sha} {mode} expanded. Special key "kind" returns the kind.
func TargetGet(env Env, cfg Config, mode, name, key string, required bool) (string, error) {
	if name == "" {
		return "", fmt.Errorf("$%s is not set: the deploy step runs once per target (clog ci targets)", TargetVar)
	}
	t, ok := cfg.Targets[name]
	if !ok {
		return "", fmt.Errorf("no ci.targets.%s in the clog config (have: %s)", name, strings.Join(sortedKeys(cfg.Targets), ", "))
	}
	if err := t.validate(name); err != nil {
		return "", err
	}
	if key == "kind" {
		return t.Kind, nil
	}
	row := t.data(mode)
	if row == nil {
		return "", fmt.Errorf("ci.targets.%s has no %s: block, so it cannot deploy in %s mode", name, mode, mode)
	}
	val := row[key]
	if isEmptyValue(val) {
		if required {
			return "", fmt.Errorf("ci.targets.%s.%s.%s is not set", name, mode, key)
		}
		return "", nil
	}
	return expandTokens(renderValue(val), env, mode), nil
}

// expandTokens replaces {tag} {version} {sha} {mode} in target data.
//
//	{tag}      release tag at HEAD, else the dev version       v0.11.4 | v0.11.4+dev.3.gabc1234
//	{version}  the same without a leading v                    0.11.4
//	{sha}      commit being built
func expandTokens(s string, env Env, mode string) string {
	if !strings.Contains(s, "{") {
		return s
	}
	version := strings.TrimPrefix(ReleaseVersion(), "v")
	tag := version
	if tag != "" {
		tag = "v" + tag
	}
	return strings.NewReplacer(
		"{version}", version,
		"{tag}", tag,
		"{sha}", commitSHA(env),
		"{mode}", mode,
	).Replace(s)
}

// commitSHA is the commit under build, from the CI environment or local git.
func commitSHA(env Env) string {
	if sha := firstNonEmpty(env.Getenv("GITHUB_SHA"), env.Getenv("CI_COMMIT_SHA")); sha != "" {
		return sha
	}
	if g, ok := env.Git.(interface{ HeadSHA() string }); ok {
		return g.HeadSHA()
	}
	return ""
}

// HeadSHA returns the full commit SHA of HEAD, or "".
func (g execGit) HeadSHA() string { return g.git("rev-parse", "HEAD") }

var (
	targetsKindFlag  string
	targetGetRequire bool
)

var targetsCmd = &cobra.Command{
	Use:          "targets [--kind <kind>]",
	Short:        "list the deploy targets used in this run's mode, one per line",
	Long:         targetsHelp,
	SilenceUsage: true,
	Args:         cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := LoadConfig()
		if err != nil {
			return err
		}
		d, err := Decide(DefaultEnv())
		if err != nil {
			return err
		}
		names, err := TargetNames(cfg, d.Mode, targetsKindFlag)
		if err != nil {
			return err
		}
		for _, n := range names {
			fmt.Fprintln(cmd.OutOrStdout(), n)
		}
		return nil
	},
}

var targetCmd = &cobra.Command{
	Use:          "target get <key>",
	Short:        "print one value from $CLOG_TARGET's data for this run's mode",
	Long:         targetsHelp,
	SilenceUsage: true,
	Args:         cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if args[0] != "get" {
			return fmt.Errorf("unknown `ci target` sub-command %q (want `get <key>`)", args[0])
		}
		cfg, err := LoadConfig()
		if err != nil {
			return err
		}
		env := DefaultEnv()
		d, err := Decide(env)
		if err != nil {
			return err
		}
		val, err := TargetGet(env, cfg, d.Mode, strings.TrimSpace(env.Getenv(TargetVar)), args[1], targetGetRequire)
		if err != nil {
			return err
		}
		if val != "" {
			fmt.Fprintln(cmd.OutOrStdout(), val)
		}
		return nil
	},
}
