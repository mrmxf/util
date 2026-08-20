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

var envCmd = &cobra.Command{
	Use:   "env [get <key>]",
	Short: "print the environment this event builds (dev|stage|prod), or one of its settings",
	Long:  envHelp,
	// SilenceErrors is deliberately NOT set here (unlike `resolve`). Neither
	// host app prints the error a RunE returns, so silencing cobra too would
	// turn "CLOG_ENV is a typo" into a bare exit 1 in a CI log. A wrong
	// environment publishes the wrong site - it has to say so out loud.
	SilenceUsage: true,
	Args:         cobra.MaximumNArgs(2),
	RunE:         runEnv,
}

func init() {
	resolveCmd.Flags().StringVar(&formatFlag, "format", "json",
		"output format: json (default) or env (KEY=value lines for $GITHUB_ENV / dotenv)")
	Command.AddCommand(resolveCmd)
	Command.AddCommand(envCmd)
}

// runEnv implements the three shapes of `clog ci env`:
//
//	clog ci env              → the environment name
//	clog ci env show         → the whole resolved row, as KEY=value lines
//	clog ci env get <key>    → one setting
//
// Names print without a trailing context so they compose directly:
//
//	hugo build --baseURL "$(clog ci env get base-url)"
func runEnv(cmd *cobra.Command, args []string) error {
	name, err := ResolveEnvName(osEnv())
	if err != nil {
		return err
	}
	out := cmd.OutOrStdout()

	switch {
	case len(args) == 0:
		_, err := fmt.Fprintln(out, name)
		return err

	case args[0] == "show":
		row, err := EnvRow(name)
		if err != nil {
			return err
		}
		for _, key := range rowKeys(row) {
			if _, err := fmt.Fprintf(out, "%s=%s\n", key, renderValue(row[key])); err != nil {
				return err
			}
		}
		return nil

	case args[0] == "get":
		if len(args) != 2 {
			return fmt.Errorf("`ci env get` needs exactly one key, e.g. `clog ci env get base-url`")
		}
		val, err := EnvGet(name, args[1])
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(out, val)
		return err

	default:
		return fmt.Errorf("unknown `ci env` sub-command %q (want `get <key>` or `show`)", args[0])
	}
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
