//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/

package install

import (
	"fmt"
	"io/fs"

	"gopkg.in/yaml.v3"
)

// LoadManifest reads index.yaml from the embedded filesystem.
func LoadManifest() (*Manifest, error) {
	data, err := EmbedFS.ReadFile("index.yaml")
	if err != nil {
		return nil, fmt.Errorf("install: cannot read embedded index.yaml: %w", err)
	}
	var m Manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("install: cannot parse index.yaml: %w", err)
	}
	if m.Tools == nil {
		m.Tools = make(map[string]ToolEntry)
	}
	return &m, nil
}

// LoadTool resolves a named tool from the manifest.
// The index entry's recipepath is resolved (embedfs-first via EmbedFS, then
// the caller-supplied userFS, then OS filesystem). Returns the source path
// alongside the resolved entry so callers can print it with --recipe.
func LoadTool(m *Manifest, name string, userFS fs.FS) (sourcePath string, entry ToolEntry, err error) {
	indexEntry, ok := m.Tools[name]
	if !ok {
		return "", ToolEntry{}, fmt.Errorf("tool %q not found in manifest (try `clog install list`)", name)
	}

	// Try the util/install embedded FS first
	sourcePath, entry, err = ResolveRecipe(name, indexEntry, EmbedFS)
	if err == nil {
		return sourcePath, entry, nil
	}

	// If a user-supplied FS was provided, try that next
	if userFS != nil {
		sourcePath, entry, err = ResolveRecipe(name, indexEntry, userFS)
		if err == nil {
			return sourcePath, entry, nil
		}
	}

	return "", ToolEntry{}, fmt.Errorf("load tool %q: %w", name, err)
}
