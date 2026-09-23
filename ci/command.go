//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/

package ci

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
)

var formatFlag string
var configHelpFlag bool

// Command is the `clog CI` cobra command. It groups CI-orchestration helpers;
// today the only sub-command is `resolve`.
var Command = &cobra.Command{
	Use:   "CI",
	Short: "CI <sub-command> - normalize CI/CD event context for build steps",
	Long:  longHelp,
	Run: func(cmd *cobra.Command, args []string) {
		if configHelpFlag {
			fmt.Fprintln(cmd.OutOrStdout(), configHelp)
			return
		}
		cmd.Help()
	},
}

var resolveCmd = &cobra.Command{
	Use:           "event",
	Short:         "resolve the active CI event into normalized ref/repo/verb",
	Long:          resolveHelp,
	SilenceErrors: true,
	SilenceUsage:  true,
	Args:          cobra.NoArgs,
	RunE:          runResolve,
}

// showCmd - `clog CI show <policy|event|mode|scan>`: prints a document.
var showCmd = &cobra.Command{
	Use:           "show",
	Short:         "print a document describing this run (JSON or KEY=value)",
	SilenceErrors: true,
	SilenceUsage:  true,
	Run:           ciHelpRun,
}

// listCmd - `clog CI list`: reserved for repo-wide lists. The per-namespace
// lists live under their namespace (`CI target list`, `CI stack list`), so
// the verb still sits immediately after the thing it acts on.
var listCmd = &cobra.Command{
	Use:           "list",
	Short:         "print zero or more values, one per line",
	SilenceErrors: true,
	SilenceUsage:  true,
	Run:           ciHelpRun,
}

func ciHelpRun(cmd *cobra.Command, args []string) {
	cmd.Help() //nolint:errcheck // help output is best-effort
}

func init() {
	resolveCmd.Flags().StringVar(&formatFlag, "format", "json",
		"output format: json (default) or env (KEY=value lines for $GITHUB_ENV / dotenv)")
	Command.Flags().BoolVar(&configHelpFlag, "config-help", false,
		"print how to configure Infisical OIDC secrets for CI (identities, subjects, .clog.yaml keys)")
	runCmd.Flags().BoolVar(&runDryRunFlag, "dry-run", false, "fetch and report secrets (names only) but do not run the command")
	runCmd.Flags().SetInterspersed(false) // flags after <command> belong to the command
	Command.AddCommand(runCmd)
	policyCmd.Flags().StringVar(&policyFormatFlag, "format", "json", "output format: json (default) or env (build_mode/deploy_mode/do_build/do_deploy/deploy_targets lines for $GITHUB_ENV)")
	getCmd.Flags().BoolVar(&getRequiredFlag, "required", false, "exit 1 when the key is missing or empty")
	Command.AddCommand(getCmd)
	Command.AddCommand(shouldCmd)
	Command.AddCommand(requireCmd)
	modeCmd.Flags().BoolVar(&modeGetRequired, "required", false, "exit 1 when the key is missing or empty")
	Command.AddCommand(modeCmd)
	Command.AddCommand(envCmd) // deprecated alias of `ci mode` (mode.go)
	targetsCmd.Flags().StringVar(&targetsKindFlag, "kind", "", "only targets of this kind (container-registry|bucket|cloudflare-pages|github-pages|gitlab-pages|github-release|gitlab-release)")
	// On targetGetCmd, not targetCmd: `get` is a real subcommand here (unlike
	// `mode get`, where get is an argument), so a flag on the parent is one the
	// only command that reads it can never be given.
	targetGetCmd.Flags().BoolVar(&targetGetRequire, "required", false, "exit 1 when the key is missing or empty")
	Command.AddCommand(targetCmd)
	deployCmd.Flags().StringVar(&deployTargetFlag, "target", "", "deploy only this ci.targets.<name>")
	deployCmd.Flags().BoolVar(&deployDryRunFlag, "dry-run", false, "report what would be published, change nothing")
	deployCmd.Flags().StringVar(&deployModeFlag, "mode", "", "force the mode (dev|prod) instead of the one ci.policy resolves")
	deployCmd.Flags().StringVar(&deployStackFlag, "stack", "", "only this ci.stack entry's targets (name, or `all`; default: all)")
	Command.AddCommand(deployCmd)
	stackCmd.PersistentFlags().StringVar(&stackSelectFlag, "stack", "", "which stack (name, or `all`; default: all, and the first one for `watch`)")
	stackEchoCmd.Flags().StringVar(&stackEchoModeFlag, "mode", "", "the mode to name in the line")
	stackEchoCmd.Flags().BoolVar(&stackEchoWatchFlag, "watch", false, "use watch selection (the first stack), not batch selection")
	Command.AddCommand(stackCmd)
	scanCmd.Flags().StringVar(&scanFormatFlag, "format", "env", "output format: env (default) or json")
	scanCmd.Flags().StringVar(&scanTargetFlag, "target", "", "resolve the artifact sweep for this ci.targets.<name> instead of the source sweep")
	scanNsCmd.PersistentFlags().StringVar(&scanModeFlag, "mode", "", "force the mode (dev|prod) instead of the one ci.policy resolves")

	scanNsCmd.PersistentFlags().StringVar(&scanStackFlag, "stack", "", "which stack (name, or `all`; default: all)")
}

func runResolve(cmd *cobra.Command, args []string) error {
	r, err := Resolve(DefaultEnv())
	if err != nil {
		return err
	}
	out := cmd.OutOrStdout()
	switch formatFlag {
	case "json":
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		return enc.Encode(r)
	case "env":
		_, err := fmt.Fprint(out, r.EnvLines())
		return err
	default:
		return fmt.Errorf("unknown --format %q (want json or env)", formatFlag)
	}
}

// EnvLines renders the resolution as shell KEY=value lines, suitable for
// appending to $GITHUB_ENV or sourcing as a dotenv file. The lowercase keys
// verb/depth/ref/repo/url match the names the legacy workflows wrote, so the
// new `clog CI show event --format env >> $GITHUB_ENV` is a drop-in replacement.
func (r Resolution) EnvLines() string {
	return "" +
		"ci=" + string(r.CI) + "\n" +
		"verb=" + string(r.Verb) + "\n" +
		"ref=" + r.Ref + "\n" +
		"repo=" + r.Repo + "\n" +
		"url=" + r.URL + "\n" +
		"depth=" + strconv.Itoa(r.Depth) + "\n" +
		"actor=" + r.Actor + "\n" +
		"is_production=" + strconv.FormatBool(r.IsProduction) + "\n" +
		"is_tag=" + strconv.FormatBool(r.IsTag) + "\n"
}
