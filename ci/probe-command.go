//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/

package ci

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	probeTargetFlag string
	probeModeFlag   string
	probeDryRunFlag bool
	probeStrictFlag bool
	probeDirFlag    string
)

// ProbeDirDefault is where a probe report lands. The name is canonical so a
// debugger looks rather than guesses, and it sits beside the deploy receipts
// because that is the run that produced it.
const ProbeDirDefault = "_clog_deploy/probe"

var probeCmd = &cobra.Command{
	Use:   "probe [--target <name>] [--mode dev|prod] [--strict]",
	Short: "ask the LIVE deploy targets whether they are exposed",
	Long: `CI probe - what is live, as opposed to what sits still.

A scan reads source, lockfiles and binaries, so it runs at build time on every
branch and pull request. A probe cannot run until something is deployed: a
bucket that is not yet public cannot be publicly listable, and a site that is
not yet served cannot be serving its own .git directory.

  clog CI probe                every target this mode deployed to
  clog CI probe --target site  just that one
  clog CI probe --strict       exit 1 on any failing finding

Findings are reported, not fatal, by default. The deploy has already happened
by the time a probe runs, so a non-zero exit cannot un-publish anything - what
it can do is say so in the same run, which is why --strict exists for pipelines
that gate a promotion on the answer.

The report is written to ` + ProbeDirDefault + `/probe.json.`,
	SilenceUsage: true,
	Args:         cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := LoadConfig()
		if err != nil {
			return err
		}
		env := DefaultEnv()
		mode := probeModeFlag
		if mode == "" {
			d, err := Decide(env)
			if err != nil {
				return err
			}
			mode = d.Mode
		}
		if mode != ModeDev && mode != ModeProd {
			return fmt.Errorf("--mode %q: want %s or %s", mode, ModeDev, ModeProd)
		}

		report, err := Probe(env, cfg, mode, probeTargetFlag, probeDryRunFlag, cmd.OutOrStdout())
		if err != nil {
			return err
		}
		FormatProbe(report, cmd.OutOrStdout())

		path, err := WriteProbeReport(report, probeDirFlag)
		if err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "probe report: %s\n", path)

		if report.Failed() && probeStrictFlag {
			// The findings are already printed; an error here would repeat
			// them under an "Error:" prefix that adds nothing.
			os.Exit(1)
		}
		return nil
	},
}

func init() {
	probeCmd.Flags().StringVar(&probeTargetFlag, "target", "", "probe only this ci.targets.<name>")
	probeCmd.Flags().StringVar(&probeModeFlag, "mode", "", "force the mode (dev|prod) instead of the one ci.policy resolves")
	probeCmd.Flags().BoolVar(&probeDryRunFlag, "dry-run", false, "report what would be probed, ask nothing")
	probeCmd.Flags().BoolVar(&probeStrictFlag, "strict", false, "exit 1 when any finding is a failure")
	probeCmd.Flags().StringVar(&probeDirFlag, "dir", ProbeDirDefault, "where to write the report")
	Command.AddCommand(probeCmd)
}
