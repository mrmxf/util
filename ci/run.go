//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/

package ci

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

// RunMarkerVar is set to 1 in the child of `clog CI run`, so a snippet can
// tell it is already running with secrets and must not re-exec itself.
const RunMarkerVar = "CLOG_CI_RUN"

// DryRun is a hook for the host app's global dry-run flag (e.g. clog --dryrun).
var DryRun func() bool

var runDryRunFlag bool

var runCmd = &cobra.Command{
	Use:   "run [--dry-run] -- <command> [args...]",
	Short: "run a command with this repo's Infisical secrets in its environment",
	Long:  runHelp,
	// A wrong or missing secret must say so in the job log (see envCmd).
	SilenceUsage: true,
	Args:         cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		code, err := Run(DefaultEnv(), args, cmd.OutOrStdout(), isRunDryRun())
		if err != nil {
			return err
		}
		if code != 0 {
			exit(code)
		}
		return nil
	},
}

// exit is os.Exit, replaceable in tests.
var exit = os.Exit

// userToken returns the laptop user's Infisical session token. A var so tests
// need no CLI.
var userToken = func(domain string) (string, error) {
	out, err := exec.Command("infisical", "user", "get", "token", "--plain").Output()
	token := strings.TrimSpace(string(out))
	if err != nil || token == "" {
		return "", fmt.Errorf("no Infisical user session: run  infisical login --domain=%s", domain)
	}
	return token, nil
}

func isRunDryRun() bool {
	return runDryRunFlag || (DryRun != nil && DryRun())
}

// Run fetches the secrets for the current CI context and runs args[0] with
// them added to its environment. It returns the child's exit code.
//
//   - pull/merge request: never fetches secrets; runs the command without them
//   - GitHub / GitLab:    OIDC login as the dev or prod identity from .clog.yaml
//   - laptop:             uses the `infisical login` user session
func Run(env Env, args []string, stdout io.Writer, dryRun bool) (int, error) {
	r, err := Resolve(env)
	if err != nil {
		return 0, err
	}
	d, err := Decide(env)
	if err != nil {
		return 0, err
	}

	var secrets []Secret
	if r.Verb == VerbPR {
		slog.Warn("pull/merge request: running without secrets", "ci", r.CI, "mode", d.Mode)
	} else {
		secrets, err = fetchForContext(env, r, d.Mode)
		if err != nil {
			return 0, err
		}
	}

	if r.CI == PlatformGitHub {
		writeGitHubMasks(stdout, secrets)
	}

	if dryRun {
		slog.Info("dry-run: not running", "command", strings.Join(args, " "))
		return 0, nil
	}
	return runChild(args, childEnv(os.Environ(), secrets))
}

// fetchForContext logs in the right way for the platform and fetches secrets.
func fetchForContext(env Env, r Resolution, mode string) ([]Secret, error) {
	cfg, err := LoadConfig()
	if err != nil {
		return nil, err
	}
	inf := cfg.Infisical
	if inf.isUnset() {
		slog.Info("no ci.infisical block: running without secrets", "mode", mode)
		return nil, nil
	}
	if err := inf.validate(); err != nil {
		return nil, err
	}
	infEnv, err := inf.infisicalEnv(mode)
	if err != nil {
		return nil, err
	}

	var token, via string
	switch r.CI {
	case PlatformGitHub, PlatformGitLab:
		key, uuid, err := inf.identityFor(r.CI, mode)
		if err != nil {
			return nil, err
		}
		jwt, err := platformJWT(env, r.CI, inf.audience())
		if err != nil {
			return nil, err
		}
		if token, err = infisicalLogin(inf.Domain, uuid, jwt); err != nil {
			return nil, err
		}
		via = fmt.Sprintf("oidc %s identity %s (%s)", r.CI, key, uuid)
	default:
		if token, err = userToken(inf.Domain); err != nil {
			return nil, err
		}
		via = "infisical user login"
	}

	secrets, err := infisicalFetch(inf.Domain, token, inf.ProjectID, infEnv, inf.Path)
	if err != nil {
		return nil, err
	}
	slog.Info("secrets loaded", "count", len(secrets), "names", secretNames(secrets),
		"mode", mode, "infisical-env", infEnv, "path", inf.Path, "via", via)
	return secrets, nil
}

// writeGitHubMasks registers every secret value (each line of multi-line
// values) with the Actions runner so it is redacted from the job log.
func writeGitHubMasks(w io.Writer, secrets []Secret) {
	for _, s := range secrets {
		for _, line := range strings.Split(s.Value, "\n") {
			if strings.TrimSpace(line) != "" {
				fmt.Fprintf(w, "::add-mask::%s\n", line)
			}
		}
	}
}

// childEnv returns base with the secrets added (a secret replaces an inherited
// variable of the same name) and RunMarkerVar=1.
func childEnv(base []string, secrets []Secret) []string {
	override := map[string]bool{RunMarkerVar: true}
	for _, s := range secrets {
		override[s.Key] = true
	}
	out := make([]string, 0, len(base)+len(secrets)+1)
	for _, kv := range base {
		name, _, _ := strings.Cut(kv, "=")
		if override[name] {
			if name != RunMarkerVar {
				slog.Warn("secret replaces an inherited environment variable", "name", name)
			}
			continue
		}
		out = append(out, kv)
	}
	for _, s := range secrets {
		out = append(out, s.Key+"="+s.Value)
	}
	return append(out, RunMarkerVar+"=1")
}

// runChild runs args with env, wiring stdio through, and returns its exit code.
func runChild(args []string, env []string) (int, error) {
	c := exec.Command(args[0], args[1:]...) //nolint:gosec // the command is the caller's to choose
	c.Env = env
	c.Stdin, c.Stdout, c.Stderr = os.Stdin, os.Stdout, os.Stderr
	err := c.Run()
	var exitErr *exec.ExitError
	switch {
	case err == nil:
		return 0, nil
	case errors.As(err, &exitErr):
		return exitErr.ExitCode(), nil
	default:
		return 0, fmt.Errorf("cannot run %q: %w", args[0], err)
	}
}

func secretNames(secrets []Secret) []string {
	names := make([]string, len(secrets))
	for i, s := range secrets {
		names[i] = s.Key
	}
	return names
}
