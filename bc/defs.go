//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

// Package bc provides BC (build-control) functionality for the clog application.
// BC (build-control) manages build processes, version control, and deployment operations
// with a focus on Git integration and tag management.
package bc

import "time"

// BCkey is the configuration key used to access BC (build-control) settings
// in the configuration system.
const BCkey = "build-control"

type StashLevels int

const (
	LvlDebug   StashLevels = iota
	LvlTrace   StashLevels = iota
	LvlInfo    StashLevels = iota
	LvlSuccess StashLevels = iota
	LvlWarn    StashLevels = iota
	// beyond here are errors
	LvlError     StashLevels = iota
	LvlFatal     StashLevels = iota
	LvlEmergency StashLevels = iota
)

// FlowPhaseStep represents a single step in a flow with its message and timestamp
type FlowPhaseStep struct {
	Step      string      `yaml:"step" json:"step"`
	Level     StashLevels `yaml:"level" json:"level"`
	Message   string      `yaml:"message" json:"message"`
	Timestamp time.Time   `yaml:"timestamp" json:"timestamp"`
}

type PhaseSteps []FlowPhaseStep

type PhaseName string

type FlowPhases map[PhaseName]PhaseSteps

type FlowName string

// Stash represents named flows with named phases that log each of its steps
type Stash struct {
	Flow map[FlowName]FlowPhases `yaml:"flow" json:"flow"`
}
