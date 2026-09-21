//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/

package ci

import (
	"fmt"
	"log/slog"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

// The two modes. There is no staging: a run builds and deploys either the dev
// shape of a project or the production one (D-I.12).
const (
	ModeDev  = "dev"
	ModeProd = "prod"
)

const (
	// ModeOverrideVar forces the mode, so a laptop can reproduce a CI run:
	//   CLOG_MODE=prod clog ci policy
	ModeOverrideVar = "CLOG_MODE"
	// LegacyModeVar is the pre-D-I.12 name, still read (with a warning) so
	// existing scripts keep working for one release.
	LegacyModeVar = "CLOG_ENV"
	// ModesKey holds per-mode BUILD settings (deploy data lives in ci.targets).
	ModesKey = "ci.modes"
)

// modeOverride returns the forced mode, or "". "stage" is rejected by name:
// silently mapping it to dev or prod would build the wrong thing.
func modeOverride(env Env) (string, error) {
	val, from := strings.TrimSpace(env.Getenv(ModeOverrideVar)), ModeOverrideVar
	if val == "" {
		if val = strings.TrimSpace(env.Getenv(LegacyModeVar)); val == "" {
			return "", nil
		}
		from = LegacyModeVar
		slog.Warn("$"+LegacyModeVar+" is deprecated: use $"+ModeOverrideVar, "value", val)
	}
	switch val {
	case ModeDev, ModeProd:
		return val, nil
	case "stage":
		return "", fmt.Errorf("$%s=stage: staging was removed - a run is %s or %s (clog ci --config-help)", from, ModeDev, ModeProd)
	default:
		return "", fmt.Errorf("$%s=%q is not %s or %s", from, val, ModeDev, ModeProd)
	}
}

// ModeData returns the ci.modes.<mode> table of build settings.
var ModeData = func(mode string) map[string]any {
	raw, ok := ConfigValue(ModesKey + "." + mode).(map[string]any)
	if !ok {
		return nil
	}
	return raw
}

// ModeGet returns one ci.modes.<mode> setting, rendered for a shell.
func ModeGet(mode, key string, required bool) (string, error) {
	val := ModeData(mode)[key]
	if isEmptyValue(val) {
		if required {
			return "", fmt.Errorf("%s.%s.%s is not set in the clog config", ModesKey, mode, key)
		}
		return "", nil
	}
	return renderValue(val), nil
}

var modeGetRequired bool

var modeCmd = &cobra.Command{
	Use:   "mode [dev|prod] [get <key>]",
	Short: "print this run's mode (dev|prod), validate a forced one, or read a ci.modes setting",
	Long:  modeHelp,
	// a wrong mode builds the wrong thing - say so out loud (see envCmd history)
	SilenceUsage: true,
	Args:         cobra.MaximumNArgs(2),
	RunE:         runMode,
}

// envCmd is the pre-D-I.12 name for `ci mode`, kept for one release.
var envCmd = &cobra.Command{
	Use:          "env [get <key>]",
	Short:        "deprecated: use `clog ci mode`",
	Long:         modeHelp,
	Hidden:       true,
	SilenceUsage: true,
	Args:         cobra.MaximumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		slog.Warn("`clog ci env` is deprecated: use `clog ci mode` (staging was removed, D-I.12)")
		return runMode(cmd, args)
	},
}

func runMode(cmd *cobra.Command, args []string) error {
	d, err := Decide(DefaultEnv())
	if err != nil {
		return err
	}
	out := cmd.OutOrStdout()

	switch {
	case len(args) == 0:
		_, err := fmt.Fprintln(out, d.Mode)
		return err

	// `clog ci mode prod` validates a forced mode and echoes it, so a snippet
	// can turn its own argument into an override in one line:
	//
	//   [ -n "$1" ] && export CLOG_MODE="$(clog ci mode "$1")"
	//
	// Sites used to carry a private bc-mode snippet for exactly this, and the
	// copies drifted. Validation belongs here, where a typo is an error rather
	// than a silent dev build.
	case args[0] == ModeDev || args[0] == ModeProd:
		if len(args) != 1 {
			return fmt.Errorf("`ci mode %s` takes no further arguments", args[0])
		}
		_, err := fmt.Fprintln(out, args[0])
		return err

	case args[0] == "get":
		if len(args) != 2 {
			return fmt.Errorf("`ci mode get` needs exactly one key, e.g. `clog ci mode get base-url`")
		}
		val, err := ModeGet(d.Mode, args[1], modeGetRequired)
		if err != nil {
			return err
		}
		if val != "" {
			fmt.Fprintln(out, val)
		}
		return nil

	case args[0] == "show":
		row := ModeData(d.Mode)
		for _, key := range sortedKeys(row) {
			if _, err := fmt.Fprintf(out, "%s=%s\n", key, renderValue(row[key])); err != nil {
				return err
			}
		}
		return nil

	default:
		return fmt.Errorf("unknown `ci mode` argument %q (want `dev`, `prod`, `get <key>` or `show`)", args[0])
	}
}

// renderValue renders a config value for a shell: scalars as-is, lists one item
// per line, so `for t in $(clog ci targets)` and `--flag "$(clog ci mode get x)"`
// both work without quoting games.
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

func sortedKeys[T any](m map[string]T) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// osEnv is the process environment, used by the cobra commands.
func osEnv() Env {
	e := DefaultEnv()
	if e.Getenv == nil {
		e.Getenv = os.Getenv
	}
	return e
}
