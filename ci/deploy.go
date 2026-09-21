//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/

package ci

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

// Deployment is everything a deployer needs to publish one target in one mode.
// It is a struct rather than a long argument list so adding a field later does
// not touch every deployer.
type Deployment struct {
	Env    Env
	Cfg    Config
	Mode   string
	Target string // ci.targets.<name>
	DryRun bool
	Out    io.Writer
}

// Get reads one key from this deployment's target data, with {tag} {version}
// {sha} {mode} already expanded.
func (d Deployment) Get(key string, required bool) (string, error) {
	return TargetGet(d.Env, d.Cfg, d.Mode, d.Target, key, required)
}

// GetOr reads a key and falls back to def when it is unset.
func (d Deployment) GetOr(key, def string) (string, error) {
	v, err := d.Get(key, false)
	if err != nil {
		return "", err
	}
	if v == "" {
		return def, nil
	}
	return v, nil
}

// Deployer publishes an already-built artefact to one destination. One per
// target kind; the kind constants are in targets.go.
type Deployer func(Deployment) error

// deployers maps a target kind to its implementation. A kind may be declared
// in knownKinds (so `clog ci targets` validates it) before a deployer exists —
// Deploy reports that as a clear error rather than skipping the target.
var deployers = map[string]Deployer{
	KindGitHubPages: deployGitHubPages,
}

// execCommand runs name in dir and returns its combined output. A package var
// so tests can exercise the deployers without git, a network or a repo.
var execCommand = func(dir, name string, args ...string) (string, error) {
	c := exec.Command(name, args...) //nolint:gosec // args are built from validated config
	c.Dir = dir
	out, err := c.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

// mkdirTemp is os.MkdirTemp, replaceable in tests.
var mkdirTemp = os.MkdirTemp

// Deploy runs the deployer for every target that deploys in mode. When only is
// set, just that target runs. Each target is independent: the first failure
// stops the run, because a half-deployed release is worse than a stopped one.
func Deploy(env Env, cfg Config, mode, only string, dryRun bool, out io.Writer) error {
	names, err := TargetNames(cfg, mode, "")
	if err != nil {
		return err
	}
	if only != "" {
		if _, ok := cfg.Targets[only]; !ok {
			return fmt.Errorf("no ci.targets.%s in the clog config (have: %s)",
				only, strings.Join(sortedKeys(cfg.Targets), ", "))
		}
		if !contains(names, only) {
			return fmt.Errorf("ci.targets.%s does not deploy in %s mode", only, mode)
		}
		names = []string{only}
	}
	if len(names) == 0 {
		slog.Warn("no deploy targets for this mode", "mode", mode)
		return nil
	}

	for _, name := range names {
		t := cfg.Targets[name]
		run, ok := deployers[t.Kind]
		if !ok {
			return fmt.Errorf("ci.targets.%s.kind %q is valid but has no deployer yet (have: %s)",
				name, t.Kind, strings.Join(sortedKeys(deployers), ", "))
		}
		slog.Info("deploy target", "target", name, "kind", t.Kind, "mode", mode, "dry-run", dryRun)
		if err := run(Deployment{Env: env, Cfg: cfg, Mode: mode, Target: name, DryRun: dryRun, Out: out}); err != nil {
			return fmt.Errorf("ci.targets.%s (%s): %w", name, t.Kind, err)
		}
	}
	return nil
}

func contains(list []string, want string) bool {
	for _, v := range list {
		if v == want {
			return true
		}
	}
	return false
}

var (
	deployTargetFlag string
	deployDryRunFlag bool
	deployModeFlag   string
)

var deployCmd = &cobra.Command{
	Use:          "deploy [--target <name>] [--mode dev|prod] [--dry-run]",
	Short:        "publish the build to every target for this run's mode",
	Long:         deployHelp,
	SilenceUsage: true,
	Args:         cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := LoadConfig()
		if err != nil {
			return err
		}
		env := DefaultEnv()
		d, err := Decide(env)
		if err != nil {
			return err
		}
		// --mode forces the mode without the caller having to export CLOG_MODE,
		// which is what `clog deploy prod` needs to mean something.
		mode := d.Mode
		if deployModeFlag != "" {
			if deployModeFlag != ModeDev && deployModeFlag != ModeProd {
				return fmt.Errorf("--mode %q: want %s or %s", deployModeFlag, ModeDev, ModeProd)
			}
			mode = deployModeFlag
			slog.Warn("deploy mode forced on the command line", "mode", mode, "resolved", d.Mode)
		}
		dry := deployDryRunFlag || (DryRun != nil && DryRun())
		return Deploy(env, cfg, mode, deployTargetFlag, dry, cmd.OutOrStdout())
	},
}
