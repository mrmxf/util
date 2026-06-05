//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/

// Package embedfs provides a minimal embed.FS containing a starter konfig.yaml
// and generic helpers for searching an embed.FS by filename.
//
// For a fork that embeds private data, create your own package that exports a
// CoreFs variable and pass it to the helper functions here.
package embedfs

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"path/filepath"
)

//go:embed konfig.yaml
var CoreFs embed.FS

// HasFilePath reports whether path exists in efs.
func HasFilePath(efs embed.FS, path string) bool {
	f, err := efs.Open(path)
	if err == nil {
		f.Close()
		return true
	}
	return false
}

// FindEmbeddedFile returns all paths within efs whose base name equals name.
func FindEmbeddedFile(efs embed.FS, name string) ([]string, error) {
	slog.Debug(fmt.Sprintf("searching for (%s) in embedded file system", name))
	return searchFolder(efs, name, nil)
}

func searchFolder(efs embed.FS, name string, matches []string) ([]string, error) {
	found := false
	err := fs.WalkDir(efs, ".", func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !d.IsDir() && filepath.Base(p) == name {
			slog.Debug("found: " + p)
			matches = append(matches, p)
			found = true
		}
		return nil
	})

	if err != nil {
		return matches, errors.New(err.Error() + " (" + name + " not found)")
	}
	if !found {
		slog.Debug("not found: " + name)
	}
	return matches, nil
}
