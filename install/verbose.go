//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/

package install

import (
	"log/slog"
	"net/http"
	"strings"

	"gopkg.in/yaml.v3"
)

// Verbose, when true, logs the resolved configuration and where each part of
// it came from at INFO level. Set by the --verbose flag.
var Verbose bool

// DryRun is a hook for the host app's global dry-run flag (e.g. `clog --dryrun`).
// The host sets it during bootstrap; nil means "no host flag".
var DryRun func() bool

// unknownVersion is the placeholder used when no version could be resolved.
const unknownVersion = "[unknown]"

// isDryRun reports whether either the local --dry-run flag or the host's
// global dry-run hook is set, and which one.
func isDryRun() (bool, string) {
	if dryRunFlag {
		return true, "--dry-run flag"
	}
	if DryRun != nil && DryRun() {
		return true, "global --dryrun flag"
	}
	return false, ""
}

// vlog logs at INFO only when Verbose is set.
func vlog(msg string, args ...any) {
	if Verbose {
		slog.Info(msg, args...)
	}
}

// vlogYAML logs label followed by a YAML dump of v, one line per record, only
// when Verbose is set.
func vlogYAML(label string, v any, args ...any) {
	if !Verbose {
		return
	}
	slog.Info(label, args...)
	out, err := yaml.Marshal(v)
	if err != nil {
		slog.Warn("  cannot marshal config for display", "err", err)
		return
	}
	for _, line := range strings.Split(strings.TrimRight(string(out), "\n"), "\n") {
		slog.Info("    " + line)
	}
}

// checkURL does a HEAD request (following redirects) so a dry-run can report
// whether the resolved URL exists without downloading it.
func checkURL(url string) {
	resp, err := http.Head(url) //nolint:gosec // URL is built from trusted config
	if err != nil {
		slog.Warn("  url check failed", "url", url, "err", err)
		return
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		slog.Warn("  url check failed", "url", url, "status", resp.StatusCode)
		return
	}
	slog.Info("  url check ok", "status", resp.StatusCode, "bytes", resp.ContentLength)
}
