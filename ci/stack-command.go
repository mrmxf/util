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

// stackCmd is the namespace: `clog CI stack <verb>`.
var stackCmd = &cobra.Command{
	Use:   "stack",
	Short: "the named build units in ci.stack",
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
	Run:          ciHelpRun,
}

// stackListCmd - `clog CI stack list [tools|chk|make|names]`: zero or more
// values. With no noun it lists the stack names themselves.
var stackListCmd = &cobra.Command{
	Use:          "list [tools|chk|make|names]",
	Short:        "print stack names, or one resolved list, one per line",
	SilenceUsage: true,
	Args:         cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := LoadConfig()
		if err != nil {
			return err
		}
		out := cmd.OutOrStdout()
		if len(args) == 0 {
			stacks, err := Stacks(cfg)
			if err != nil {
				return err
			}
			for _, st := range stacks {
				fmt.Fprintln(out, st.Name)
			}
			return nil
		}
		selected, _, err := StackSelect(cfg, stackSelectFlag)
		if err != nil {
			return err
		}
		var vals []string
		switch args[0] {
		case "tools":
			vals = StackTools(selected)
		case "chk":
			vals = StackChk(selected)
		case "make":
			vals = StackMake(selected)
		case "names":
			vals = StackNames(selected)
		default:
			return fmt.Errorf("unknown `CI stack list` key %q (want tools, chk, make or names)", args[0])
		}
		// A list prints one value per line so `for x in $(...)` and `read`
		// both work. The space-joined form the dispatcher used could not
		// carry a value containing a space.
		for _, v := range vals {
			fmt.Fprintln(out, v)
		}
		return nil
	},
}

// stackGetCmd - `clog CI stack get <watch|type>`: exactly one value.
var stackGetCmd = &cobra.Command{
	Use:          "get <watch|type>",
	Short:        "print one resolved setting for a single stack",
	SilenceUsage: true,
	Args:         cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := LoadConfig()
		if err != nil {
			return err
		}
		out := cmd.OutOrStdout()
		switch args[0] {
		case "watch":
			st, _, err := StackWatch(cfg, stackSelectFlag)
			if err != nil {
				return err
			}
			// An unset value prints nothing, not a blank line: the `get`
			// contract is that an empty capture means unset.
			if st.Watch != "" {
				fmt.Fprintln(out, st.Watch)
			}
			return nil
		case "type":
			selected, _, err := StackSelect(cfg, stackSelectFlag)
			if err != nil {
				return err
			}
			if len(selected) != 1 {
				return fmt.Errorf("`CI stack get type` needs one stack (--stack <name>); this repo has %s",
					strings.Join(StackNames(selected), ", "))
			}
			fmt.Fprintln(out, selected[0].Type)
			return nil
		default:
			return fmt.Errorf("unknown `CI stack get` key %q (want watch or type)", args[0])
		}
	},
}

// stackEchoCmd - `clog CI stack echo <verb>`: the banner a verb prints.
var stackEchoCmd = &cobra.Command{
	Use:          "echo <verb>",
	Short:        "print the one-line banner naming what a verb is about to act on",
	SilenceUsage: true,
	Args:         cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := LoadConfig()
		if err != nil {
			return err
		}
		out := cmd.OutOrStdout()
		if stackEchoWatchFlag {
			chosen, all, err := StackWatch(cfg, stackSelectFlag)
			if err != nil {
				return err
			}
			fmt.Fprintln(out, StackEcho(args[0], []Stack{chosen}, all, stackEchoModeFlag))
			return nil
		}
		selected, all, err := StackSelect(cfg, stackSelectFlag)
		if err != nil {
			return err
		}
		fmt.Fprintln(out, StackEcho(args[0], selected, all, stackEchoModeFlag))
		return nil
	},
}
