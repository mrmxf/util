//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

package kfg

import (
	"errors"
	"log/slog"
	"os"
)

// AutoMerge reads the "kfg" configuration section to get a list of paths to merge,
// then attempts to merge each configuration file found in those paths.
//
// How it works:
//   - 1. Uses Unmarshal to extract the "kfg" section into a KonfigPaths struct
//   - 2. Iterates through each path in the TryPaths array
//   - 3. For each path, attempts to merge the configuration file using MergeKonfig
//   - 4. Logs success or failure for each file using slog.Debug
//   - 5. Missing files are silently ignored (merge is optional)
//
// Returns an error only if the "kfg" section cannot be unmarshaled or if there's
// a critical configuration error. File-not-found errors are ignored.
func AutoMerge() error {
	// Check if Konfigure() has been called first
	if Raw == nil {
		return errors.New("configuration not initialized: call Konfigure() before using AutoMerge()")
	}

	// Extract the kfg configuration section to get merge paths
	konfigPaths, err := Unmarshal[KonfigPaths]("kfg", "koanf")
	if err != nil {
		slog.Debug("AutoMerge: failed to unmarshal kfg: yaml", "error", err)
		return err
	}

	if konfigPaths == nil {
		slog.Debug("AutoMerge: no kfg section found in configuration")
		return nil
	}

	slog.Debug("AutoMerge: starting ", "count", len(konfigPaths.TryPaths))

	// Iterate through each path and attempt to merge
	for i, path := range konfigPaths.TryPaths {
		if path == "" {
			slog.Debug("AutoMerge: skipping empty path", "index", i)
			continue
		}

		// slog.Debug("AutoMerge: attempting to merge", "path", path, "index", i)

		// Use MergeKonfig with custom path
		MergeKonfig(&KonfigureOpt{
			AppFs:           os.DirFS("."), // Use OS filesystem for merge paths
			FilePath:        path,
			PreventAutoLoad: false,
		})

	}

	slog.Debug("AutoMerge: completed automatic merge", "total_paths", len(konfigPaths.TryPaths))
	return nil
}
