//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/

package ci

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/mrmxf/util/kfg"
)

// The three environment names. They are deliberately few: an environment is a
// deployment target, not a feature flag. Adding a fourth should mean a fourth
// place a build can land, not a variation on an existing one.
const (
	EnvDev   = "dev"
	EnvStage = "stage"
	EnvProd  = "prod"
)

// EnvOverrideVar lets a developer force an environment locally, so the staging
// or production shape of a build can be reproduced on a laptop:
//
//	CLOG_ENV=stage clog build
//
// It is honoured in CI too. That is intentional — a one-off manual re-run of a
// pipeline sometimes needs to target a different environment, and the override
// is visible in the job log rather than hidden in a branch name.
const EnvOverrideVar = "CLOG_ENV"

// EnvironmentsKey is the config key holding the per-environment table. It lives
// in the consuming repo's .clog.yaml, because the VALUES (a URL, an image tag)
// are site-specific while the code reading them is not.
const EnvironmentsKey = "environments"

// Environments is an overridable hook returning the environment table, in the
// same style as the bc package's config hooks: the default reads the merged
// clog config, a host app may repoint it, and tests inject fixtures.
var Environments = func() map[string]map[string]any {
	if kfg.Raw == nil {
		return nil
	}
	raw, ok := kfg.Raw.Get(EnvironmentsKey).(map[string]any)
	if !ok {
		return nil
	}
	out := make(map[string]map[string]any, len(raw))
	for name, row := range raw {
		if r, ok := row.(map[string]any); ok {
			out[name] = r
		}
	}
	return out
}

// EnvName maps a resolved CI event to an environment name.
//
// The ordering of the cases is the whole logic, so it is worth stating plainly:
//
//   - a laptop build and a pull/merge request are ALWAYS dev. They must never
//     be able to reach a deployment target, whatever ref they happen to carry.
//     This case is first precisely so a PR raised against a tag cannot fall
//     through to prod.
//   - a scheduled run (IsProduction) or a tag push is prod. The nightly
//     re-renders the production tag so date-gated content publishes itself.
//   - everything else — a branch push, a manual dispatch — is stage.
func EnvName(r Resolution) string {
	switch {
	case r.Verb == VerbLocal, r.Verb == VerbPR:
		return EnvDev
	case r.IsProduction, r.IsTag:
		return EnvProd
	default:
		return EnvStage
	}
}

// ResolveEnvName returns the environment for the current context, honouring the
// CLOG_ENV override. The override is validated against the environments table
// when one is configured, so a typo fails loudly instead of silently building
// the wrong site.
func ResolveEnvName(env Env) (string, error) {
	if forced := strings.TrimSpace(env.Getenv(EnvOverrideVar)); forced != "" {
		if table := Environments(); table != nil {
			if _, ok := table[forced]; !ok {
				return "", fmt.Errorf("%s=%q is not one of the configured environments (%s)",
					EnvOverrideVar, forced, strings.Join(envNames(table), ", "))
			}
		}
		return forced, nil
	}
	r, err := Resolve(env)
	if err != nil {
		return "", err
	}
	return EnvName(r), nil
}

// EnvRow returns the configured row for one environment.
func EnvRow(name string) (map[string]any, error) {
	table := Environments()
	if len(table) == 0 {
		return nil, fmt.Errorf("no %q table in the clog config - add one to .clog.yaml", EnvironmentsKey)
	}
	row, ok := table[name]
	if !ok {
		return nil, fmt.Errorf("environment %q is not configured (have: %s)",
			name, strings.Join(envNames(table), ", "))
	}
	return row, nil
}

// EnvGet returns one value from an environment row, rendered for a shell.
//
// Scalars print as-is. Lists print one item per line, which is what makes both
// of these work without quoting games:
//
//	for tag in $(clog ci env get image-tags); do ...
//	clog ci env get image-tags | while IFS= read -r tag; do ...
//
// A configured-but-empty value returns an empty string with no error: an empty
// hugo-flags or an empty image-tags list is a legitimate production setting.
func EnvGet(name, key string) (string, error) {
	row, err := EnvRow(name)
	if err != nil {
		return "", err
	}
	val, ok := row[key]
	if !ok {
		return "", fmt.Errorf("environment %q has no key %q (have: %s)",
			name, key, strings.Join(rowKeys(row), ", "))
	}
	return renderValue(val), nil
}

// renderValue flattens a config value into the shell-facing form described on
// EnvGet. Nested maps are not supported on purpose — an environment row is a
// flat table of settings, and keeping it flat is what keeps it reviewable.
func renderValue(val any) string {
	switch v := val.(type) {
	case nil:
		return ""
	case string:
		return v
	case []any:
		parts := make([]string, 0, len(v))
		for _, item := range v {
			parts = append(parts, fmt.Sprint(item))
		}
		return strings.Join(parts, "\n")
	case []string:
		return strings.Join(v, "\n")
	default:
		return fmt.Sprint(v)
	}
}

func envNames(table map[string]map[string]any) []string {
	names := make([]string, 0, len(table))
	for name := range table {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func rowKeys(row map[string]any) []string {
	keys := make([]string, 0, len(row))
	for k := range row {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// osEnv is the process environment, used by the cobra command.
func osEnv() Env {
	e := DefaultEnv()
	if e.Getenv == nil {
		e.Getenv = os.Getenv
	}
	return e
}
