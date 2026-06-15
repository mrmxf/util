//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

package check

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/mrmxf/util/kfg"
)

// runDump implements the `--dump section.id [--conditions a,b,c]` CLI path.
// It parses the flag value, loads the section from kfg, and prints the block.
func runDump(_ string) error {
	section, idx, err := parseDumpTarget(dumpFlag)
	if err != nil {
		return err
	}
	indices, err := parseConditionIndices(conditionsFlag)
	if err != nil {
		return err
	}
	return DumpBlock(section, idx, indices)
}

// parseDumpTarget splits "section.id" on the LAST dot; id is a 1-based int.
func parseDumpTarget(s string) (section string, idx int, err error) {
	dot := strings.LastIndex(s, ".")
	if dot < 0 {
		return "", 0, fmt.Errorf("--dump value %q must be section.id", s)
	}
	section = s[:dot]
	idx, err = strconv.Atoi(s[dot+1:])
	if err != nil {
		return "", 0, fmt.Errorf("--dump id in %q is not an integer: %w", s, err)
	}
	return section, idx, nil
}

// parseConditionIndices parses a comma-separated list of 1-based indices.
// An empty string yields a nil slice (meaning: dump the whole block).
func parseConditionIndices(s string) ([]int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	parts := strings.Split(s, ",")
	out := make([]int, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		n, err := strconv.Atoi(p)
		if err != nil {
			return nil, fmt.Errorf("--conditions %q: %q is not an integer", s, p)
		}
		out = append(out, n)
	}
	return out, nil
}

// DumpBlock prints a check block for post-mortem inspection. With no condition
// indices it prints the whole block definition; with indices it prints, one
// line per index, the condition text at each 1-based position. Output goes to
// stdout so it can be captured from finally: or a shell.
func DumpBlock(section string, blockIdx int, conditionIndices []int) error {
	groupKey := KfgKey + "." + section
	raw := kfg.Raw.Get(groupKey)
	if raw == nil {
		return fmt.Errorf("cannot find check section(%s)", groupKey)
	}
	s, err := ParseSection(section, raw)
	if err != nil {
		return fmt.Errorf("cannot parse check section(%s): %w", groupKey, err)
	}
	if blockIdx < 1 || blockIdx > len(s.Blocks) {
		return fmt.Errorf("block index %d out of range for section %q (1..%d)", blockIdx, section, len(s.Blocks))
	}
	b := s.Blocks[blockIdx-1]

	// build the ordered condition list: env first, then try (matches eval order).
	conds := make([]Condition, 0, len(b.Env)+len(b.Try))
	kinds := make([]string, 0, len(b.Env)+len(b.Try))
	for _, c := range b.Env {
		conds = append(conds, c)
		kinds = append(kinds, "env")
	}
	for _, c := range b.Try {
		conds = append(conds, c)
		kinds = append(kinds, "try")
	}

	out := os.Stdout

	if len(conditionIndices) == 0 {
		// whole-block dump
		fmt.Fprintf(out, "block %q (%d/%d) in section %q\n", b.Name, blockIdx, len(s.Blocks), section)
		for i, c := range conds {
			fmt.Fprintf(out, "  %d: %s %q %s\n", i+1, kinds[i], c.Expr, c.State)
		}
		if b.Then != "" {
			fmt.Fprintf(out, "  then: %s\n", b.Then)
		}
		if b.Else != "" {
			fmt.Fprintf(out, "  else: %s\n", b.Else)
		}
		if b.Finally != "" {
			fmt.Fprintf(out, "  finally: %s\n", b.Finally)
		}
		return nil
	}

	// selected conditions only
	for _, idx := range conditionIndices {
		if idx < 1 || idx > len(conds) {
			fmt.Fprintf(out, "%d: (no such condition; block has %d)\n", idx, len(conds))
			continue
		}
		c := conds[idx-1]
		fmt.Fprintf(out, "%d: %s %q %s\n", idx, kinds[idx-1], c.Expr, c.State)
	}
	return nil
}
