//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

package kfg

import (
	"errors"
	"log/slog"
	"os"
	"strings"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/fs"
)

// MergeKonfig merges additional configuration from a filesystem into the existing global koanf instance.
// This allows overlaying user-specific configuration on top of the base configuration loaded by Konfigure.
//
// Parameters:
//   - opt: Optional configuration options. If not provided, uses default values:
//   - ConfigFs: OS filesystem
//   - FilePath: KonfigFilePath ("./.clog.yaml")
//   - AutoLoad: true
//
// How it works:
//   - 1. Checks that Kfg has been initialized by Konfigure first
//   - 2. Applies default values for any unspecified options
//   - 3. Uses fs.Provider to read from the specified filesystem (defaults to OS filesystem)
//   - 4. Uses yaml.Parser() to parse the YAML content
//   - 5. Merges the parsed data into the existing global koanf instance (Kfg)
//
// Returns an error if Kfg is not initialized, file cannot be read, or parsing fails.
// Returns nil if successful or if the file doesn't exist (merge is optional).
func MergeKonfig(opt ...*KonfigureOpt) error {
	// Check if Konfigure() has been called first
	if Raw == nil {
		return errors.New("configuration not initialized: call Konfigure() before using MergeKonfig()")
	}

	// Apply defaults if no opt provided or opt is nil
	var options *KonfigureOpt
	if len(opt) == 0 || opt[0] == nil {
		options = &KonfigureOpt{}
	} else {
		options = opt[0]
	}

	configFs := options.AppFs
	if configFs == nil {
		configFs = os.DirFS(".") // Default to OS filesystem at current directory
	}

	filePath := options.FilePath
	if filePath == "" {
		filePath = KonfigFilePath
	}

	if options.PreventAutoLoad {
		return nil
	}

	// Try to load the file, but don't fail if it doesn't exist (merge is optional)
	err := Raw.Load(fs.Provider(configFs, filePath), yaml.Parser())

	// Check for various file not found or access errors, which are acceptable for merge
	switch {
	case err == nil:
		slog.Debug("AutoMerge: konfig search", "found", true, "path", filePath)
		return nil
	case os.IsNotExist(err):
		slog.Debug("AutoMerge: konfig search", "found", false, "path", filePath)
		return nil
	case strings.Contains(err.Error(), "file does not exist"):
		slog.Debug("AutoMerge: konfig search", "found", false, "path", filePath)
		return nil
	case strings.Contains(err.Error(), "no such file"):
		slog.Debug("AutoMerge: konfig search", "found", false, "path", filePath)
		return nil
	case strings.Contains(err.Error(), "invalid argument"):
		slog.Debug("AutoMerge: konfig search", "found", false, "path", filePath)
		return nil
	case strings.Contains(err.Error(), "not found"):
		slog.Debug("AutoMerge: konfig search", "found", false, "path", filePath)
		return nil
	default:
		return err // Some other error occurred
	}
}
