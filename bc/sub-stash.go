//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

package bc

import (
	"os"
	"runtime"

	slog "github.com/mrmxf/util/slogger"

	"github.com/spf13/cobra"
)

// stashCmd check the stash log for conditions
var stashCmd = &cobra.Command{
	Use:           "stash",
	SilenceErrors: true,
	SilenceUsage:  true,
	Short:         "check or extract information from the stash",
	Long:          stashLongHelp,
	Example:       stashExample,
}

// stashCmd check the stash log for conditions
var stashHasCmd = &cobra.Command{
	Use:           "has",
	SilenceErrors: true,
	SilenceUsage:  true,
	Short:         "check or extract information from the stash",
	Long:          stashLongHelp,
	Example:       stashExample,
	Run:           stashHasRun,
}

// stashCmd check the stash log for conditions
var stashGetCmd = &cobra.Command{
	Use:           "get",
	SilenceErrors: true,
	SilenceUsage:  true,
	Short:         "get information from the stash",
	Long:          stashLongHelp,
	Example:       stashExample,
}

// stashGetErrorCmd gets the most recent error from the stash
var stashGetErrorCmd = &cobra.Command{
	Use:           "error",
	SilenceErrors: true,
	SilenceUsage:  true,
	Short:         "get the most recent error from the stash",
	Long:          "Prints the error with the highest timestamp, optionally filtered by flow",
	Example:       "clog bc stash get error --flow build",
	Run:           stashGetErrorFunc,
}

func init() {
	_, file, _, _ := runtime.Caller(0)
	slog.Debug("init " + file)

	// Stash-specific flags
	stashCmd.PersistentFlags().StringVarP(&slFlow, "flow", "1", "", "flow name for stash organization")
	stashCmd.PersistentFlags().StringVarP(&slPhase, "phase", "2", "", "phase name for stash organization")
	stashCmd.PersistentFlags().StringVarP(&slStep, "step", "3", "", "step name (optional, prepended to message)")

	// Add subcommands to the main BC command
	stashCmd.AddCommand(stashHasCmd)
	stashCmd.AddCommand(stashGetCmd)
	stashGetCmd.AddCommand(stashGetErrorCmd)
	Command.AddCommand(stashCmd)
}

// stashHasRun checks the stash for errors and exits with appropriate code
func stashHasRun(cmd *cobra.Command, args []string) {
	// Load the stash using the library function
	stash := LoadStash()

	// Track if we found any errors (Level > LvlWarn)
	hasError := false
	errorInSpecifiedFlow := false

	// Parse the stash map to check for errors
	for flowName, flowPhases := range stash.Flow {
		for _, phaseSteps := range flowPhases {
			for _, step := range phaseSteps {
				if step.Level > LvlWarn {
					hasError = true
					// Check if this error is in the specified flow
					if slFlow != "" && string(flowName) == slFlow {
						errorInSpecifiedFlow = true
					}
				}
			}
		}
	}

	// Exit logic based on conditions:
	// 1. If Level > LvlWarn and no flow specified - exit with 1
	if hasError && slFlow == "" {
		os.Exit(1)
	}

	// 2. If Level > LvlWarn and a flowName was specified with -1/--flow flag - exit with 1 if error found in that flow
	if slFlow != "" && errorInSpecifiedFlow {
		os.Exit(1)
	}

	// Otherwise exit 0
	os.Exit(0)
}

// stashGetErrorFunc gets and prints the most recent error from the stash
func stashGetErrorFunc(cmd *cobra.Command, args []string) {
	// Load the stash using the library function
	stash := LoadStash()

	var mostRecentError *FlowPhaseStep
	var mostRecentFlow FlowName
	var mostRecentPhase PhaseName

	// Parse the stash map to find the error with the highest timestamp
	for flowName, flowPhases := range stash.Flow {
		// If a flow is specified, only check that flow
		if slFlow != "" && string(flowName) != slFlow {
			continue
		}

		for phaseName, phaseSteps := range flowPhases {
			for _, step := range phaseSteps {
				if step.Level > LvlWarn {
					// Check if this is the most recent error
					if mostRecentError == nil || step.Timestamp.After(mostRecentError.Timestamp) {
						// Make a copy of the step to avoid pointer issues
						stepCopy := step
						mostRecentError = &stepCopy
						mostRecentFlow = flowName
						mostRecentPhase = phaseName
					}
				}
			}
		}
	}

	// Print the most recent error if found
	if mostRecentError != nil {
		slog.Info("Stashed Error",
			"flow", mostRecentFlow,
			"phase", mostRecentPhase,
			"step", mostRecentError.Step,
			"message", mostRecentError.Message,
			"timestamp", mostRecentError.Timestamp.Format("2006-01-02 15:04:05"))
	}
}
