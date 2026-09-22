//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/

package ci

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var (
	stackSelectFlag    string
	stackEchoModeFlag  string
	stackEchoWatchFlag bool
)

var stackCmd = &cobra.Command{
	Use:   "stack [get <tools|watch|chk|make>] [echo <verb>] [--stack <name|all>]",
	Short: "list this repo's stacks, or read one resolved setting",
	Long: `ci stack - the named build units in ci.stack.

With no arguments it lists every stack name, in declaration order. The first is
the default for ` + "`clog watch`" + `, so order is meaningful.

  clog ci stack                       every name, one per line
  clog ci stack get tools             tools for every stack, space separated
  clog ci stack get make --stack bonfire

The selector follows the verbs: empty or "all" means every stack, a name means
that one. ` + "`tools`" + `, ` + "`chk`" + ` and ` + "`make`" + ` are unioned across the selected
stacks in declaration order and deduplicated, so a phase named by two stacks
runs once. ` + "`watch`" + ` cannot be unioned and needs a single stack.`,
	SilenceUsage: true,
	Args:         cobra.RangeArgs(0, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := LoadConfig()
		if err != nil {
			return err
		}
		out := cmd.OutOrStdout()

		if len(args) == 2 && args[0] == "echo" {
			verb := args[1]
			if stackEchoWatchFlag {
				chosen, all, err := StackWatch(cfg, stackSelectFlag)
				if err != nil {
					return err
				}
				fmt.Fprintln(out, StackEcho(verb, []Stack{chosen}, all, stackEchoModeFlag))
				return nil
			}
			selected, all, err := StackSelect(cfg, stackSelectFlag)
			if err != nil {
				return err
			}
			fmt.Fprintln(out, StackEcho(verb, selected, all, stackEchoModeFlag))
			return nil
		}

		if len(args) == 0 {
			stacks, err := Stacks(cfg)
			if err != nil {
				return err
			}
			for _, s := range stacks {
				fmt.Fprintln(out, s.Name)
			}
			return nil
		}
		if args[0] != "get" || len(args) != 2 {
			return fmt.Errorf("unknown `ci stack` sub-command %q (want `get <tools|watch|chk|make>` or `echo <verb>`)", strings.Join(args, " "))
		}

		key := args[1]
		if key == "watch" {
			s, _, err := StackWatch(cfg, stackSelectFlag)
			if err != nil {
				return err
			}
			fmt.Fprintln(out, s.Watch)
			return nil
		}
		selected, _, err := StackSelect(cfg, stackSelectFlag)
		if err != nil {
			return err
		}
		var vals []string
		switch key {
		case "tools":
			vals = StackTools(selected)
		case "chk":
			vals = StackChk(selected)
		case "make":
			vals = StackMake(selected)
		case "names":
			vals = StackNames(selected)
		case "type":
			if len(selected) != 1 {
				return fmt.Errorf("`ci stack get type` needs one stack (--stack <name>); this repo has %s",
					strings.Join(StackNames(selected), ", "))
			}
			vals = []string{selected[0].Type}
		default:
			return fmt.Errorf("unknown `ci stack get` key %q (want tools, watch, chk, make, names or type)", key)
		}
		if len(vals) > 0 {
			fmt.Fprintln(out, strings.Join(vals, " "))
		}
		return nil
	},
}
