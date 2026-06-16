//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

package bc

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

// SemVer represents a semantic version with major, minor, and patch components
type SemVer struct {
	Major int
	Minor int
	Patch int
}

// parseSemVer parses a semantic version string into a SemVer struct
func parseSemVer(name string, version string) (SemVer, error) {
	// Remove 'v' prefix if present
	version = strings.TrimPrefix(version, "v")

	parts := strings.Split(version, ".")
	if len(parts) != 3 {
		return SemVer{}, fmt.Errorf("invalid semver (%s): %s", name, version)
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return SemVer{}, fmt.Errorf("invalid major version (%s): %s", name, parts[0])
	}

	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return SemVer{}, fmt.Errorf("invalid minor version (%s): %s", name, parts[1])
	}

	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		return SemVer{}, fmt.Errorf("invalid patch version (%s): %s", name, parts[2])
	}

	return SemVer{
		Major: major,
		Minor: minor,
		Patch: patch,
	}, nil
}

// semverCmd provides BC (build-control) semantic version comparison operations.
var semverCmd = &cobra.Command{
	Use:   "semver",
	Short: "BC semver  <needs> <installed> - version comparison",
	Long: `BC (build-control) semantic version comparison validates version compatibility:
- Compares two semantic version strings
- Ensures major versions match
- Validates minor and patch version requirements
- Exits with error if compatibility requirements are not met

Arguments:
  needs      The required semantic version (e.g., "1.2.3" or "v1.2.3")
  installed  The currently installed semantic version (e.g., "1.2.3" or "v1.2.3")

Examples:
  clog BC semver  1.2.3   1.2.3    # Valid: exact match
  clog BC semver v1.2.0   1.3.0    # Valid: newer minor version (v is trimmed)
  clog BC semver  1.2.3   1.2.5    # Valid: newer patch version
  clog BC semver  1.2.3   1.2.2    # Warning: older patch version
  clog BC semver  1.2.3   1.1.0    # Error: older minor version
  clog BC semver  1.2.3   2.0.0    # Error: different major version`,
	Args:          cobra.ExactArgs(2),
	SilenceErrors: true,
	SilenceUsage:  true,
	Run: func(cmd *cobra.Command, args []string) {
		needsStr := args[0]
		installedStr := args[1]

		needs, err := parseSemVer("needs", needsStr)
		if err != nil {
			slog.Error("Invalid param #1", "version", needsStr, "error", err)
			os.Exit(1)
		}

		installed, err := parseSemVer("installed", installedStr)
		if err != nil {
			slog.Error("Invalid param #2", "version", installedStr, "error", err)
			os.Exit(1)
		}

		// Check if major versions are the same
		if needs.Major != installed.Major {
			slog.Error("Major version mismatch",
				"needs_major", needs.Major,
				"installed_major", installed.Major,
				"needs", needsStr,
				"installed", installedStr)
			os.Exit(1)
		}

		// Check if installed minor version is less than needed
		if installed.Minor < needs.Minor {
			slog.Error("Installed minor version is too old",
				"needs_minor", needs.Minor,
				"installed_minor", installed.Minor,
				"needs", needsStr,
				"installed", installedStr)
			os.Exit(1)
		}

		// Check if same minor version but patch is too old
		if installed.Minor == needs.Minor && installed.Patch < needs.Patch {
			slog.Warn("Installed patch version is older than needed",
				"needs_patch", needs.Patch,
				"installed_patch", installed.Patch,
				"needs", needsStr,
				"installed", installedStr)
		}

		// If we reach here, the version compatibility is acceptable
		slog.Debug("Version compatibility check passed",
			"needs", needsStr,
			"installed", installedStr)
	},
}

func init() {
	// Add semver subcommand to the main BC command
	Command.AddCommand(semverCmd)
}
