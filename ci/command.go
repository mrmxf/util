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

// Command is the `clog ci` cobra command. It groups CI-orchestration helpers;
// today the only sub-command is `resolve`.
var Command = &cobra.Command{
	Use:   "ci",
	Short: "ci <sub-command> - normalize CI/CD event context for build steps",
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
	Use:           "resolve",
	Short:         "resolve the active CI event into normalized ref/repo/verb",
	Long:          resolveHelp,
	SilenceErrors: true,
	SilenceUsage:  true,
	Args:          cobra.NoArgs,
	RunE:          runResolve,
}

func init() {
	resolveCmd.Flags().StringVar(&formatFlag, "format", "json",
		"output format: json (default) or env (KEY=value lines for $GITHUB_ENV / dotenv)")
	Command.Flags().BoolVar(&configHelpFlag, "config-help", false,
		"print how to configure Infisical OIDC secrets for CI (identities, subjects, .clog.yaml keys)")
	runCmd.Flags().BoolVar(&runDryRunFlag, "dry-run", false, "fetch and report secrets (names only) but do not run the command")
	runCmd.Flags().SetInterspersed(false) // flags after <command> belong to the command
	Command.AddCommand(resolveCmd)
	Command.AddCommand(runCmd)
	policyCmd.Flags().StringVar(&policyFormatFlag, "format", "json", "output format: json (default) or env (build_mode/deploy_mode/do_build/do_deploy/deploy_targets lines for $GITHUB_ENV)")
	Command.AddCommand(policyCmd)
	getCmd.Flags().BoolVar(&getRequiredFlag, "required", false, "exit 1 when the key is missing or empty")
	Command.AddCommand(getCmd)
	Command.AddCommand(shouldCmd)
	Command.AddCommand(requireCmd)
	modeCmd.Flags().BoolVar(&modeGetRequired, "required", false, "exit 1 when the key is missing or empty")
	Command.AddCommand(modeCmd)
	Command.AddCommand(envCmd) // deprecated alias of `ci mode` (mode.go)
	targetsCmd.Flags().StringVar(&targetsKindFlag, "kind", "", "only targets of this kind (container-registry|bucket|package|cloudflare-pages|github-pages|gitlab-pages)")
	Command.AddCommand(targetsCmd)
	targetCmd.Flags().BoolVar(&targetGetRequire, "required", false, "exit 1 when the key is missing or empty")
	Command.AddCommand(targetCmd)
	deployCmd.Flags().StringVar(&deployTargetFlag, "target", "", "deploy only this ci.targets.<name>")
	deployCmd.Flags().BoolVar(&deployDryRunFlag, "dry-run", false, "report what would be published, change nothing")
	Command.AddCommand(deployCmd)
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
// new `clog ci resolve --format env >> $GITHUB_ENV` is a drop-in replacement.
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
