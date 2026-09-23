//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/

package bc

import "github.com/mrmxf/util/kfg"

// StashPathKey is the config key holding the stash file path, and
// StashPathDefault is where it lands when the key is unset. The default is
// under the canonical build artifact directory: the stash is a record of what
// the check phases found, which is build output like any other.
const (
	StashPathKey     = "clog.stash-path"
	StashPathDefault = "_clog_build/check/stash.yaml"
)

// Config hooks decouple bc from any single app's config struct (it used to read
// the clog-mrmxf `my.App`). Each is an overridable function with a sensible
// default sourced from util/kfg. A host app may override them at bootstrap to
// point at its own config; tests override them to inject fixtures.
var (
	// Releases returns the loaded release list (current release first).
	// Defaults to the kfg release cache populated at boot.
	Releases = func() []kfg.AppRelease { return kfg.Releases() }

	// ReleasesPath returns the path of the releases.yaml file (for messages).
	ReleasesPath = func() string { return kfg.ReleasesPath() }

	// StashPath returns the configured stash file path, from `clog.stash-path`.
	// Empty means "use the default", StashPathDefault.
	//
	// This used to return "" unconditionally, so `clog.stash-path` was config
	// that parsed, documented itself and did nothing - the stash always landed
	// on the hard-coded default however the key was set.
	StashPath = func() string {
		if kfg.Raw == nil {
			return ""
		}
		return kfg.Raw.String(StashPathKey)
	}

	// DryRun reports whether the host app is in dry-run mode.
	DryRun = func() bool { return false }
)
