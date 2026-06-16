//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

package bc

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"gopkg.in/yaml.v3"
)

// LoadStash reads the stash file and returns a Stash struct
// If the file doesn't exist, returns an empty Stash with initialized Flow map
func LoadStash() Stash {
	stashPath := getStashPath()

	// Check if file exists
	if _, err := os.Stat(stashPath); os.IsNotExist(err) {
		slog.Debug("stash file does not exist, returning empty stash", "path", stashPath)
		return Stash{
			Flow: make(map[FlowName]FlowPhases),
		}
	}

	// Read the file
	data, err := os.ReadFile(stashPath)
	if err != nil {
		slog.Error("failed to read stash file", "path", stashPath, "error", err)
		return Stash{
			Flow: make(map[FlowName]FlowPhases),
		}
	}

	// Parse YAML
	var stash Stash
	if err := yaml.Unmarshal(data, &stash); err != nil {
		slog.Error("failed to unmarshal stash file", "path", stashPath, "error", err)
		return Stash{
			Flow: make(map[FlowName]FlowPhases),
		}
	}

	// Ensure Flow map is initialized
	if stash.Flow == nil {
		stash.Flow = make(map[FlowName]FlowPhases)
	}

	slog.Debug("loaded stash", "path", stashPath, "flows", len(stash.Flow))
	return stash
}

// UpdateStash writes the stash data to the stash file, creating or overwriting it
func UpdateStash(stashData Stash) error {
	stashPath := getStashPath()

	// Ensure directory exists
	stashDir := filepath.Dir(stashPath)
	if err := os.MkdirAll(stashDir, 0755); err != nil {
		slog.Error("failed to create stash directory", "dir", stashDir, "error", err)
		return fmt.Errorf("failed to create stash directory: %w", err)
	}

	// Marshal to YAML
	data, err := yaml.Marshal(stashData)
	if err != nil {
		slog.Error("failed to marshal stash data", "error", err)
		return fmt.Errorf("failed to marshal stash data: %w", err)
	}

	// Write to file
	if err := os.WriteFile(stashPath, data, 0600); err != nil {
		slog.Error("failed to write stash file", "path", stashPath, "error", err)
		return fmt.Errorf("failed to write stash file: %w", err)
	}

	slog.Debug("updated stash", "path", stashPath, "flows", len(stashData.Flow))
	return nil
}

// ansiRegex is a compiled regex pattern for matching ANSI escape sequences
// Matches sequences starting with either \x1b (ESC) or \x1e
var ansiRegex = regexp.MustCompile(`[\x1b\x1e]\[[0-9;]*[a-zA-Z]`)

// stripAnsi removes ANSI escape sequences from a string
func stripAnsi(s string) string {
	return ansiRegex.ReplaceAllString(s, "")
}

// AppendStash loads the stash, appends a new step to the specified flow/phase, and updates the file
func AppendStash(flow, phase, step string, stashLevel StashLevels, msg string) error {
	// Strip ANSI escape sequences from all string parameters
	flow = stripAnsi(flow)
	phase = stripAnsi(phase)
	step = stripAnsi(step)
	msg = stripAnsi(msg)

	// Run the load→modify→write under an advisory lock so concurrent stashLog
	// processes cannot lose each other's entries (R1).
	return withStashLock(getStashPath(), func() error {
		// Load existing stash
		stashData := LoadStash()

		flowName := FlowName(flow)
		phaseName := PhaseName(phase)

		// Get or create the flow phases
		flowPhases, exists := stashData.Flow[flowName]
		if !exists {
			flowPhases = make(FlowPhases)
			stashData.Flow[flowName] = flowPhases
		}

		// Get or create the phase steps
		phaseSteps, exists := flowPhases[phaseName]
		if !exists {
			phaseSteps = make(PhaseSteps, 0)
		}

		// Append new step with timestamp
		newStep := FlowPhaseStep{
			Step:      step,
			Level:     stashLevel,
			Message:   msg,
			Timestamp: time.Now(),
		}
		phaseSteps = append(phaseSteps, newStep)

		// Update phase and flow
		flowPhases[phaseName] = phaseSteps
		stashData.Flow[flowName] = flowPhases

		// Save to file
		if err := UpdateStash(stashData); err != nil {
			return fmt.Errorf("failed to append to stash: %w", err)
		}

		slog.Debug("appended to stash", "flow", flow, "phase", phase, "message", msg)
		return nil
	})
}

// ResetStash creates a fresh empty stash file, ensuring all directories exist
func ResetStash() error {
	stashPath := getStashPath()

	// Ensure directory exists
	stashDir := filepath.Dir(stashPath)
	if err := os.MkdirAll(stashDir, 0755); err != nil {
		slog.Error("failed to create stash directory", "dir", stashDir, "error", err)
		return fmt.Errorf("failed to create stash directory: %w", err)
	}

	// Create empty stash with initialized Flow map
	emptyStash := Stash{
		Flow: make(map[FlowName]FlowPhases),
	}

	// Marshal to YAML
	data, err := yaml.Marshal(emptyStash)
	if err != nil {
		slog.Error("failed to marshal empty stash", "error", err)
		return fmt.Errorf("failed to marshal empty stash: %w", err)
	}

	// Write to file (overwrite existing)
	if err := os.WriteFile(stashPath, data, 0600); err != nil {
		slog.Error("failed to write stash file", "path", stashPath, "error", err)
		return fmt.Errorf("failed to write stash file: %w", err)
	}

	slog.Info("reset stash", "path", stashPath)
	return nil
}

// getStashPath returns the absolute path to the stash file
func getStashPath() string {
	stashPath := StashPath()
	if stashPath == "" {
		stashPath = "tmp/ClogBcStash.yaml"
	}

	absPath, err := filepath.Abs(stashPath)
	if err != nil {
		slog.Error("failed to get absolute stash path", "path", stashPath, "error", err)
		return stashPath
	}

	return absPath
}
