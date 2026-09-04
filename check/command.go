//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

//
// package check creates a try/then/else/finally block of scripts

package check

import (
	"fmt"
	"log/slog"
	"os"
	"runtime"

	"github.com/mrmxf/util/kfg"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var KfgKey = "check"

// ValidBlockFields lists every YAML key that is valid inside a check block.
var ValidBlockFields = map[string]bool{
	"name":    true,
	"before":  true,
	"env":     true,
	"try":     true,
	"then":    true,
	"ok":      true,
	"else":    true,
	"catch":   true,
	"finally": true,
}

// ParseError describes a single structural error found during --dry-run
// validation of a check block. All fields are populated so callers can
// pinpoint section / block-index / block-name / field without further lookup.
type ParseError struct {
	Section string // check group key (e.g. "tools")
	Block   int    // 1-based block index within the group
	ID      string // block name if set, otherwise "block #N"
	Field   string // YAML field that triggered the error, or "" for structural errors
	Reason  string // human-readable explanation
}

func (e ParseError) Error() string {
	if e.Field == "" {
		return fmt.Sprintf("check.%s %s: %s", e.Section, e.ID, e.Reason)
	}
	return fmt.Sprintf("check.%s %s .%s: %s", e.Section, e.ID, e.Field, e.Reason)
}

// DryRun validates the raw blocks array for a check group without executing
// any scripts. It accumulates all parse errors rather than stopping at the
// first one, so a single pass surfaces every structural problem in the config.
//
// Returns nil when the blocks are structurally valid.
func DryRun(section string, rawBlocksArray any) []ParseError {
	if rawBlocksArray == nil {
		return []ParseError{{Section: section, Block: 0, ID: "blocks", Field: "", Reason: "nil blocks array in config"}}
	}

	blocks, ok := rawBlocksArray.([]any)
	if !ok {
		return []ParseError{{Section: section, Block: 0, ID: "blocks", Field: "", Reason: fmt.Sprintf("expected array, got %T", rawBlocksArray)}}
	}

	var errs []ParseError
	for i, raw := range blocks {
		blockNum := i + 1
		blk, isMap := raw.(map[string]interface{})
		if !isMap {
			errs = append(errs, ParseError{
				Section: section, Block: blockNum, ID: fmt.Sprintf("block #%d", blockNum),
				Field: "", Reason: fmt.Sprintf("must be a YAML map, got %T", raw),
			})
			continue
		}

		// resolve a human-readable ID for this block
		id := fmt.Sprintf("block #%d", blockNum)
		if nameVal, has := blk["name"]; has {
			if s, isStr := nameVal.(string); isStr && s != "" {
				id = fmt.Sprintf("%q (#%d)", s, blockNum)
			}
		}

		// flag unrecognised keys
		for k := range blk {
			if !ValidBlockFields[k] {
				errs = append(errs, ParseError{Section: section, Block: blockNum, ID: id, Field: k, Reason: "unrecognised field"})
			}
		}

		// scalar string fields must be strings.
		for _, f := range []string{"name", "then", "ok", "else", "catch", "finally", "before"} {
			val, exists := blk[f]
			if !exists || val == nil {
				continue
			}
			if _, isStr := val.(string); !isStr {
				errs = append(errs, ParseError{
					Section: section, Block: blockNum, ID: id,
					Field: f, Reason: fmt.Sprintf("expected string, got %T", val),
				})
			}
		}

		// env / try must be a string (legacy) or a list.
		for _, f := range []string{"env", "try"} {
			val, exists := blk[f]
			if !exists || val == nil {
				continue
			}
			switch val.(type) {
			case string, []any, []string:
				// acceptable forms
			default:
				errs = append(errs, ParseError{
					Section: section, Block: blockNum, ID: id,
					Field: f, Reason: fmt.Sprintf("expected string or list, got %T", val),
				})
			}
		}

		// round-trip through YAML to catch type / schema incompatibilities,
		// including the env/try union and the then/ok + else/catch alias clashes.
		yamlBody, err := yaml.Marshal(blk)
		if err != nil {
			errs = append(errs, ParseError{Section: section, Block: blockNum, ID: id, Field: "", Reason: "yaml.Marshal: " + err.Error()})
			continue
		}
		var b Block
		if err := yaml.Unmarshal(yamlBody, &b); err != nil {
			errs = append(errs, ParseError{Section: section, Block: blockNum, ID: id, Field: "", Reason: err.Error()})
		}
	}
	return errs
}

var (
	dryRunFlag     bool
	dumpFlag       string
	conditionsFlag string
)

var Command = &cobra.Command{
	Use:           "Check",
	Short:         "run all blocks in a check group defined in config",
	Long:          longHelp,
	SilenceErrors: true,
	SilenceUsage:  true,
	RunE: func(cmd *cobra.Command, args []string) error {

		// --dump carries its own "section.id" target, so it works with or
		// without a positional section argument and short-circuits everything.
		if dumpFlag != "" {
			return runDump("")
		}

		// check a group was specified
		if len(args) == 0 {
			slog.Error("cannot run Check, no group specified", "len(args)", 0)
			ListGroups(os.Stdout)
			return cmd.Help()
		}

		if kfg.Raw.Get(KfgKey) == nil {
			slog.Error("cannot find check key in clog.yaml", "key", KfgKey)
			return fmt.Errorf("cannot find check key(%s) in clog.yaml", KfgKey)
		}

		// check which group we are checking
		groupKey := KfgKey + "." + args[0]
		if kfg.Raw.Get(groupKey) == nil {
			slog.Error("cannot find check group in clog.yaml", "key", groupKey)
			ListGroups(os.Stdout)
			return fmt.Errorf("cannot find check group(%s) in clog.yaml", groupKey)
		}

		blocksKey := groupKey + ".blocks"

		// --dry-run: validate YAML structure without executing any scripts.
		if dryRunFlag {
			parseErrs := DryRun(args[0], kfg.Raw.Get(blocksKey))
			if len(parseErrs) == 0 {
				slog.Info("Check --dry-run success - no errors", "section", args[0])
				return nil
			}
			for _, e := range parseErrs {
				slog.Error(e.Error())
			}
			return fmt.Errorf("Check --dry-run: %d parse error(s) in section %q", len(parseErrs), args[0])
		}

		section, err := ParseSection(args[0], kfg.Raw.Get(groupKey))
		if err != nil {
			slog.Error("cannot parse check section", "key", groupKey, "err", err)
			return fmt.Errorf("cannot parse check section(%s): %w", groupKey, err)
		}
		return RunSection(section)
	},
}

// ParseSection converts the raw kfg value for a check.<section> key into a
// typed Section, applying the env/try normalisation and then/else aliasing.
func ParseSection(name string, raw any) (Section, error) {
	if raw == nil {
		return Section{}, fmt.Errorf("nil section")
	}
	yamlBody, err := yaml.Marshal(raw)
	if err != nil {
		return Section{}, fmt.Errorf("yaml.Marshal: %w", err)
	}
	var s Section
	if err := yaml.Unmarshal(yamlBody, &s); err != nil {
		return Section{}, err
	}
	if s.Name == "" {
		s.Name = name
	}
	return s, nil
}

func init() {
	Command.Flags().BoolVar(&dryRunFlag, "dry-run", false, "validate YAML structure without executing scripts; reports all parse errors")
	Command.Flags().StringVar(&dumpFlag, "dump", "", "inspect a block: --dump section.id (1-based); no scripts run")
	Command.Flags().StringVar(&conditionsFlag, "conditions", "", "with --dump: comma-separated 1-based condition indices to explain")
}

func init() {
	_, file, _, _ := runtime.Caller(0)
	slog.Debug("init " + file)
}
