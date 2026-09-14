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
	req, ok := cfg.Require[verb]
	if !ok {
		slog.Debug("no ci.require entry", "verb", verb)
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
		fix := fmt.Sprintf("secrets %s: run it with secrets:  clog ci run -- clog %s", strings.Join(missingEnv, ", "), verb)
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
	clogEnv, err := ResolveEnvName(env)
	if err != nil {
		return "Infisical"
	}
	infEnv := inf.Env[clogEnv]
	if infEnv == "" {
		infEnv = "ci.infisical.env." + clogEnv + " unset"
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
