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

// Command is the `clog ci` cobra command. It groups CI-orchestration helpers;
// today the only sub-command is `resolve`.
var Command = &cobra.Command{
	Use:   "ci",
	Short: "ci <sub-command> - normalize CI/CD event context for build steps",
	Long:  longHelp,
	Run: func(cmd *cobra.Command, args []string) {
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
	Command.AddCommand(resolveCmd)
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
