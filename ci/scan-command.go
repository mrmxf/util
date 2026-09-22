//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/

package ci

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

var (
	scanFormatFlag      string
	scanTargetFlag      string
	scanModeFlag        string
	scanStackFlag       string
	scanListTargetsFlag bool
)

var scanCmd = &cobra.Command{
	Use:   "scan [--target <name>] [--format env|json]",
	Short: "print the resolved security sweep for the worktree, or for one target",
	Long: `ci scan - what to examine, not which tool examines it.

Two axes, because vulnerabilities and secrets are independent questions:

  clog ci scan                     the source sweep: the worktree
  clog ci scan --target registry   the artifact sweep: what that target ships

The source sweep runs in every repo, with or without targets. That is the axis
covering a Go library, a Hugo asset pipeline and a TinyGo module graph - three
repo shapes whose deployable has nothing scannable in it at all.

An artifact sweep resolves most-specific-first: the target's own scan block,
then ci.scan.artifact, then the target kind's default. Kinds whose output has no
dependencies - the Pages kinds, bucket - have no default, and the target must
write ` + "`scan: {vuln: none}`" + ` rather than have silence mean "not scanned".`,
	SilenceUsage: true,
	Args:         cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := LoadConfig()
		if err != nil {
			return err
		}
		env := DefaultEnv()
		mode := scanModeFlag
		if mode == "" {
			d, err := Decide(env)
			if err != nil {
				return err
			}
			mode = d.Mode
		}

		// --list-targets drives the artifact sweep loop in a check block. It
		// deliberately lists prod-only targets on a dev build: `ci targets`
		// filters by deploy-mode membership, and enumerating scans that way
		// would leave exactly those targets unscanned on every pull request.
		if scanListTargetsFlag {
			selected, all, err := StackSelect(cfg, scanStackFlag)
			if err != nil {
				return err
			}
			names, err := ScanTargets(cfg, selected, all)
			if err != nil {
				return err
			}
			for _, n := range names {
				fmt.Fprintln(cmd.OutOrStdout(), n)
			}
			return nil
		}

		axis, prefix := ScanAxis{}, "scan_source"
		if scanTargetFlag == "" {
			axis, err = SourceScan(cfg)
		} else {
			axis, prefix = ScanAxis{}, "scan_artifact"
			axis, err = ArtifactScan(env, cfg, mode, scanTargetFlag)
		}
		if err != nil {
			return err
		}

		out := cmd.OutOrStdout()
		switch scanFormatFlag {
		case "env":
			_, err := fmt.Fprint(out, axis.EnvLines(prefix))
			return err
		case "json":
			enc := json.NewEncoder(out)
			enc.SetIndent("", "  ")
			return enc.Encode(axis)
		default:
			return fmt.Errorf("unknown --format %q (want env or json)", scanFormatFlag)
		}
	},
}
