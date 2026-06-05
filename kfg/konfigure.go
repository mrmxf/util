//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

package kfg

import (
	"log/slog"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/fs"
	"github.com/knadh/koanf/v2"
)

// Raw is the global koanf instance used for configuration management. koanf is
// a configuration library that can load and parse various formats. The "."
// delimiter means nested keys are separated by dots (e.g.,
// "clog.env.aws.access"). This instance is shared across the package and holds
// all loaded configuration data.
var Raw *koanf.Koanf

// Konfigure (re)loads the configuration from a filesystem into the global koanf instance.
// This function must be called before using Unmarshal to ensure configuration data is available.
//
// Parameters:
//   - opt: Optional configuration options. If not provided, uses default values:
//   - ConfigFs: Efs (embedded filesystem)
//   - FilePath: RootConfig ("konfig.yaml")
//   - AutoLoad: true
//
// How it works:
//   - 1. Applies default values for any unspecified options
//   - 2. Initializes the global Kfg instance
//   - 3. Uses fs.Provider to read from the specified filesystem
//   - 4. Uses yaml.Parser() to parse the YAML content into a structured format
//   - 5. Loads the parsed data into the global koanf instance (Kfg)
//
// Returns an error if the file cannot be read or parsed, nil if successful.
func Konfigure(opt ...*KonfigureOpt) error {
	// Apply defaults if no opt provided or opt is nil
	var options *KonfigureOpt
	if len(opt) == 0 || opt[0] == nil {
		options = &DefaultKonfigureOpts
	} else {
		options = opt[0]
	}
	setSlogLevelDebugIfDebugFlag(options)

	configFs := options.AppFs
	if configFs == nil {
		configFs = Efs
	}

	filePath := options.FilePath
	if filePath == "" {
		filePath = RootConfig
	}

	// Initialize the global koanf instance
	Raw = koanf.New(".")

	// Load configuration if AutoLoad is true
	if options.PreventAutoLoad {
		slog.Debug("kfg.Konfigure AutoLoad is false, returning nil.")
		return nil
	}

	err := Raw.Load(fs.Provider(configFs, filePath), yaml.Parser())
	if err != nil {
		slog.Debug("kfg.Konfigure failed to load embedded "+filePath, "err", err)
		return err
	}

	// If AutoMerge is enabled, automatically merge additional configuration files
	if !options.PreventAutoMerge {
		if err := AutoMerge(); err != nil {
			slog.Debug("kfg.Konfigure failed to merge configs "+filePath, "err", err)
			return err
		}
	}

	if !options.PreventAutoApp {
		if err = AutoAppUnmarshal(options); err != nil {
			slog.Debug("kfg.Konfigure failed to unmarshal App with key "+options.AutoAppKey, "err", err)
			return err
		}
	}

	// autoloading releases does not return an error because typing `clog -v` may not be in a
	// project dir. If you need to error when Releases fails then do a manual check.
	if !options.PreventAutoReleases {
		if err := LoadReleases(options.AutoReleaseSlice); err != nil {
			slog.Debug("kfg.Konfigure failed to unmarshal release info ", "err", err)
			return nil
		}
	}

	return nil
}
