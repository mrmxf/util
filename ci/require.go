//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/

package ci

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/spf13/cobra"
)

var requireCmd = &cobra.Command{
	Use:          "require <verb>",
	Short:        "fail early if the secrets or config a verb needs (ci.require.<verb>) are missing",
	Long:         requireHelp,
	SilenceUsage: true,
	Args:         cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return Require(DefaultEnv(), args[0])
	},
}

// Require checks ci.require.<verb>: every `env` name set and non-empty, every
// `config` key non-empty. Missing `optional` names only warn. Errors name the
// variable or key and where to fix it, never a value.
func Require(env Env, verb string) error {
	cfg, err := LoadConfig()
	if err != nil {
		return err
	}
	req := cfg.Require[verb]
	// the target being deployed adds its own secrets (D-I.12): one deploy run
	// per target, each checked for exactly what that destination needs.
	target := strings.TrimSpace(env.Getenv(TargetVar))
	if t, ok := cfg.Targets[target]; ok {
		req.Env = append(append([]string(nil), req.Env...), t.Require...)
	}
	if len(req.Env)+len(req.Optional)+len(req.Config) == 0 {
		slog.Debug("nothing required", "verb", verb, "target", target)
		return nil
	}

	where := secretsLocation(env, cfg.Infisical)
	var missingEnv, missingConfig []string
	for _, name := range req.Env {
		if strings.TrimSpace(env.Getenv(name)) == "" {
			missingEnv = append(missingEnv, name)
			slog.Error("required secret not set", "name", name, "verb", verb, "source", where)
		}
	}
	for _, name := range req.Optional {
		if strings.TrimSpace(env.Getenv(name)) == "" {
			slog.Warn("optional secret not set", "name", name, "verb", verb, "source", where)
		}
	}
	for _, key := range req.Config {
		if isEmptyValue(ConfigValue(key)) {
			missingConfig = append(missingConfig, key)
			slog.Error("required config not set", "key", key, "verb", verb, "file", ".clog.yaml")
		}
	}
	if len(missingEnv)+len(missingConfig) == 0 {
		return nil
	}

	var fixes []string
	if len(missingEnv) > 0 {
		// the verb is a ci.require key, not necessarily a clog command
		// (deploy-form, say), so only build/deploy get a literal command
		cmd := "<the command that needs them>"
		if verb == "build" || verb == "deploy" {
			cmd = "clog " + verb
		}
		fix := fmt.Sprintf("secrets %s (they come from %s): run under  clog ci run -- %s",
			strings.Join(missingEnv, ", "), where, cmd)
		if env.Getenv(RunMarkerVar) == "1" {
			fix = fmt.Sprintf("secrets %s: add them in %s", strings.Join(missingEnv, ", "), where)
		}
		fixes = append(fixes, fix)
	}
	if len(missingConfig) > 0 {
		fixes = append(fixes, fmt.Sprintf("config %s: set in .clog.yaml", strings.Join(missingConfig, ", ")))
	}
	return fmt.Errorf("%s: missing %s", verb, strings.Join(fixes, "; "))
}

// secretsLocation describes where this context's secrets come from, for
// messages. It never fails: an incomplete config just gives a vaguer answer.
func secretsLocation(env Env, inf InfisicalConfig) string {
	d, err := Decide(env)
	if err != nil {
		return "Infisical"
	}
	infEnv := inf.Env[d.Mode]
	if infEnv == "" {
		infEnv = "ci.infisical.env." + d.Mode + " unset"
	}
	return fmt.Sprintf("Infisical env=%s path=%s", infEnv, inf.Path)
}

func isEmptyValue(v any) bool {
	switch t := v.(type) {
	case nil:
		return true
	case string:
		return strings.TrimSpace(t) == ""
	case []any:
		return len(t) == 0
	case map[string]any:
		return len(t) == 0
	default:
		return false
	}
}
