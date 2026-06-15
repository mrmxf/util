//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/

package install

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ResolveRecipe loads the full ToolEntry for a named tool.
// If entry.RecipePath is empty the entry is returned as-is (inline recipe).
// Otherwise the path is resolved with embedfs-first semantics:
//   - bare paths (no ./ or / prefix) → try efs first, then OS filesystem
//   - relative paths starting with ./ or ../ → OS filesystem only
//   - absolute paths → OS filesystem only
//
// Full-replace semantics: the resolved entry replaces entry entirely.
func ResolveRecipe(name string, entry ToolEntry, efs fs.FS) (string, ToolEntry, error) {
	if entry.RecipePath == "" {
		return "inline", entry, nil
	}

	path := entry.RecipePath

	// embedfs-first for bare paths
	if efs != nil && !filepath.IsAbs(path) && !strings.HasPrefix(path, "./") && !strings.HasPrefix(path, "../") {
		data, err := fs.ReadFile(efs, path)
		if err == nil {
			resolved, parseErr := parseRecipeData(name, path, data)
			return "embedfs:" + path, resolved, parseErr
		}
	}

	// OS filesystem fallback
	data, err := os.ReadFile(path)
	if err != nil {
		return "", ToolEntry{}, fmt.Errorf("recipe %q for tool %q: not found in embedfs or OS filesystem", path, name)
	}
	resolved, parseErr := parseRecipeData(name, path, data)
	return "file:" + path, resolved, parseErr
}

// parseRecipeData parses a recipe YAML file and extracts the named tool's entry.
// Recipe files use the format { toolname: ToolEntry }.
func parseRecipeData(name, path string, data []byte) (ToolEntry, error) {
	var wrapper map[string]ToolEntry
	if err := yaml.Unmarshal(data, &wrapper); err != nil {
		return ToolEntry{}, fmt.Errorf("recipe %q: cannot parse YAML: %w", path, err)
	}
	entry, ok := wrapper[name]
	if !ok {
		keys := make([]string, 0, len(wrapper))
		for k := range wrapper {
			keys = append(keys, k)
		}
		return ToolEntry{}, fmt.Errorf("recipe %q: expected key %q, found %v", path, name, keys)
	}
	return entry, nil
}
