//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/

package bc

import "github.com/mrmxf/util/kfg"

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

	// StashPath returns the configured stash file path. Empty string means
	// "use the default" (tmp/ClogBcStash.yaml).
	StashPath = func() string { return "" }

	// DryRun reports whether the host app is in dry-run mode.
	DryRun = func() bool { return false }
)
