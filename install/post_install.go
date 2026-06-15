//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/

package install

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// RunPostInstall executes post-install steps in order.
// A failed step is warned but does not abort subsequent steps;
// the first error encountered is returned after all steps run.
func RunPostInstall(steps []PostStep) error {
	var firstErr error
	for i, step := range steps {
		if err := runPostStep(step); err != nil {
			slog.Warn("post-install step failed", "step", i+1, "strategy", step.Strategy, "err", err)
			if firstErr == nil {
				firstErr = err
			}
		}
	}
	return firstErr
}

func runPostStep(step PostStep) error {
	switch step.Strategy {
	case "path-append":
		return runPathAppend(step.Line, step.Profile)
	case "go-install":
		return runGoInstall(step.Package)
	default:
		return fmt.Errorf("unknown post-install strategy %q", step.Strategy)
	}
}

// runPathAppend appends line to profile when the line is not already present
// (idempotent). Profile supports leading ~/ expansion.
func runPathAppend(line, profile string) error {
	if line == "" {
		return fmt.Errorf("path-append: line field is required")
	}
	if profile == "" {
		profile = "~/.bashrc"
	}
	if strings.HasPrefix(profile, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("path-append: get home dir: %w", err)
		}
		profile = filepath.Join(home, profile[2:])
	}

	// Idempotency: skip if the exact line is already in the file.
	if data, err := os.ReadFile(profile); err == nil {
		if strings.Contains(string(data), line) {
			slog.Info("path-append: already present", "profile", profile)
			return nil
		}
	}

	f, err := os.OpenFile(profile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("path-append: open %s: %w", profile, err)
	}
	defer f.Close()

	if _, err := fmt.Fprintf(f, "\n# added by clog install\n%s\n", line); err != nil {
		return fmt.Errorf("path-append: write to %s: %w", profile, err)
	}
	slog.Info("path-append: added", "profile", profile, "line", line)
	return nil
}
