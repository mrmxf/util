//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

package check

import (
	"fmt"
	"io"

	"github.com/mrmxf/util/kfg"
)

// GroupInfo is one discovered check group. BlockCount is -1 when the group's
// blocks: key is missing or is not a list (malformed, as distinct from empty).
type GroupInfo struct {
	Name       string
	BlockCount int
}

// Groups enumerates the check groups found in config, sorted by name. It does
// not use ParseSection so that a malformed group is still listed rather than
// aborting the whole listing.
func Groups() []GroupInfo {
	if kfg.Raw == nil {
		return nil
	}
	// MapKeys is sorted and yields an empty slice when the key is absent or
	// is not a map, so no further guard is needed.
	names := kfg.Raw.MapKeys(KfgKey)
	groups := make([]GroupInfo, 0, len(names))
	for _, name := range names {
		g := GroupInfo{Name: name, BlockCount: -1}
		if blocks, ok := kfg.Raw.Get(KfgKey + "." + name + ".blocks").([]any); ok {
			g.BlockCount = len(blocks)
		}
		groups = append(groups, g)
	}
	return groups
}

// FormatGroups writes the group listing to out, one copy-pastable command per
// line, and returns the number of groups printed.
func FormatGroups(out io.Writer, groups []GroupInfo) int {
	fmt.Fprintf(out, ">>> check groups in config key `%s`\n\n", KfgKey)

	if len(groups) == 0 {
		fmt.Fprintf(out, "  (none found — add a `%s:` section to your clog.yaml)\n\n", KfgKey)
		return 0
	}

	width := 0
	for _, g := range groups {
		if len(g.Name) > width {
			width = len(g.Name)
		}
	}
	for _, g := range groups {
		fmt.Fprintf(out, "  clog Check %-*s  %s\n", width, g.Name, blockCountLabel(g.BlockCount))
	}
	fmt.Fprintln(out)
	return len(groups)
}

// ListGroups prints the discovered check groups to out.
func ListGroups(out io.Writer) int {
	return FormatGroups(out, Groups())
}

// blockCountLabel renders a block count, pluralised, flagging malformed groups.
func blockCountLabel(n int) string {
	switch {
	case n < 0:
		return "no `blocks:` key"
	case n == 1:
		return "1 block"
	default:
		return fmt.Sprintf("%d blocks", n)
	}
}
