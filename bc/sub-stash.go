//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

package bc

import (
	"fmt"
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
	Args:          cobra.NoArgs,
	Run:           helpRun,
}

// stashCmd check the stash log for conditions
var stashHasCmd = &cobra.Command{
	Use:           "has",
	SilenceErrors: true,
	SilenceUsage:  true,
	Short:         "check or extract information from the stash",
	Long:          stashLongHelp,
	Example:       stashExample,
	// `error` is the only thing a stash can be asked to have. Without this a
	// typo (`has errors`) silently answered the error question instead.
	Args:      cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	ValidArgs: []string{"error"},
	Run:       stashHasRun,
}

// stashCmd check the stash log for conditions
var stashGetCmd = &cobra.Command{
	Use:           "get",
	SilenceErrors: true,
	SilenceUsage:  true,
	Short:         "get information from the stash",
	Long:          stashLongHelp,
	Example:       stashExample,
	Args:          cobra.NoArgs,
	Run:           helpRun,
}

// stashGetErrorCmd gets the most recent error from the stash
var stashGetErrorCmd = &cobra.Command{
	Use:           "error",
	SilenceErrors: true,
	SilenceUsage:  true,
	Short:         "get the most recent error from the stash",
	Long:          "Prints the error with the highest timestamp, optionally filtered by flow",
	Example:       "clog BC stash get error --flow build",
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

	// `has error` is a predicate, so it exits 0 when the answer is yes.
	//
	// Before v1.0.0 this was inverted - it exited 1 when the stash DID hold an
	// error - which made `if clog BC stash has error; then` run the recovery
	// branch exactly when there was nothing to recover from. The old spelling
	// is retired rather than quietly flipped, because an exit code that
	// changes meaning under an unchanged name cannot announce itself.
	if slFlow != "" {
		if errorInSpecifiedFlow {
			os.Exit(0)
		}
		os.Exit(1)
	}
	if hasError {
		os.Exit(0)
	}
	os.Exit(1)
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

	// A `get` returns one value on STDOUT, so `x=$(clog BC stash get error)`
	// captures it. Before v1.0.0 this printed through slog, which writes to
	// stderr, so the capture was always empty and the caller could not tell an
	// absent error from a present one. The structured detail is still logged,
	// because it is useful to a human reading the job output - it is just not
	// the value.
	if mostRecentError == nil {
		// Nothing found prints nothing: the `get` contract is that an unset
		// value is an empty capture, never a sentinel string.
		return
	}
	slog.Debug("Stashed Error",
		"flow", mostRecentFlow,
		"phase", mostRecentPhase,
		"step", mostRecentError.Step,
		"timestamp", mostRecentError.Timestamp.Format("2006-01-02 15:04:05"))
	fmt.Fprintln(cmd.OutOrStdout(), mostRecentError.Message)
}
