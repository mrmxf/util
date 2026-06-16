//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

package bc

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/mrmxf/util/check"
	"github.com/mrmxf/util/scripts"
	"github.com/spf13/cobra"
)

const checkFlow = "check"
const buildFlow = "build"
const summaryFlow = "FLOW"
const endPhase = "end"
const startStep = "start"
const statusStep = "status"
const endStep = "end"

// It runs check steps followed by make steps, accumulating errors and status
var flowCmd = &cobra.Command{
	Use:   "flow",
	Short: "BC flow - run check and make steps with error accumulation",
	Long: `BC flow runs a series of check and make steps based on flags or environment variables.
	

The flow command:
1. Runs check steps (from --check flag or $CHK environment variable)
2. Runs build steps (from --build flag or $MAKE environment variable)
3. Accumulates errors across all steps
4. Stashes status for downstream use
5. Aborts on error if in production mode

Flags:
  --check  - Space-separated list of check steps to run (overrides $CHK)
  --build  - Space-separated list of build steps to run (overrides $MAKE)

Environment Variables:
  CHK   - Space-separated list of check steps to run (used if --check not set)
  MAKE  - Space-separated list of make steps to run (used if --build not set)

Example:
  clog BC flow --check "lint test" --build "build deploy"
  export CHK="lint test"
  export MAKE="build deploy"
  clog BC flow

Exit codes:
- 0: all steps completed successfully
- 1: one or more steps failed`,
	SilenceErrors: true,
	SilenceUsage:  true,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get check and build steps from flags or environment
		chkSteps, buildSteps, reset := flowCliParams(cmd)
		releaseFlow := Releases()[0].Flow
		releaseBuild := Releases()[0].Build

		if reset {
			ResetStash()
			// if nothing else to do then exit
			if len(chkSteps)+len(buildSteps) == 0 {
				return nil
			}
		}
		// Get project name from environment
		project := os.Getenv("PROJECT")
		if project == "" {
			if wd, err := os.Getwd(); err == nil {
				project = filepath.Base(wd)
			} else {
				project = "project"
			}
		}

		// Log start message
		msg := fmt.Sprintf("🤖 %s %s%v %s%v release(flow:%s, build:%s)", summaryFlow, checkFlow, chkSteps, buildFlow, buildSteps, releaseFlow, releaseBuild)
		slog.Info(msg)
		AppendStash(summaryFlow, checkFlow, startStep, LvlInfo, msg)

		errorCount := 0

		// Run check steps
		for n, phase := range chkSteps {
			slog.Info(fmt.Sprintf("🤖 %s %s(%d %v)", summaryFlow, checkFlow, n+1, phase))
			AppendStash(summaryFlow, checkFlow, phase, LvlInfo, msg)

			if err := runCheck(phase); err != nil {
				errorCount++
				slog.Error(fmt.Sprintf("🤖 %s %s(%d %v)", summaryFlow, checkFlow, n+1, phase), "error", err)
				AppendStash(summaryFlow, checkFlow, phase, LvlError, err.Error())
			} else {
				AppendStash(summaryFlow, checkFlow, phase, LvlSuccess, "✅ success")
			}

			logDivider()
		}

		// Abort if checks failed in production
		switch {
		case flowIsProduction() && errorCount > 0:
			msg = fmt.Sprintf("🤖 %s %s(%v) ❌ %d check errors - aborting production build", summaryFlow, checkFlow, endStep, errorCount)
			AppendStash(summaryFlow, checkFlow, endStep, LvlError, msg)
			return fmt.Errorf("%s", msg)
		case errorCount > 0:
			msg = fmt.Sprintf("🤖 %s %s(%v) ❌ %d check errors - continuing flow(%s) build(%s)", summaryFlow, checkFlow, endStep, errorCount,
				Releases()[0].Flow, Releases()[0].Build)
			slog.Warn(msg)
			AppendStash(summaryFlow, checkFlow, endStep, LvlError, msg)
		}

		logDivider()

		var err error
		// Run make steps
		for n, phase := range buildSteps {
			slog.Info(fmt.Sprintf("🤖 %s %s(%d %v)", summaryFlow, buildFlow, n+1, phase))
			AppendStash(summaryFlow, buildFlow, phase, LvlInfo, msg)

			if err = runMakeStep(buildFlow, phase); err != nil {
				errorCount++
				slog.Error(fmt.Sprintf("🤖 %s %s(%d %v)", summaryFlow, buildFlow, n+1, phase), "error", err)
				AppendStash(summaryFlow, buildFlow, phase, LvlError, err.Error())
			} else {
				AppendStash(summaryFlow, buildFlow, phase, LvlSuccess, "✅ success")
			}

			// Abort if build failed in production
			if flowIsProduction() && errorCount > 0 {
				slog.Error("build failed", "phase", phase, "error", err)
				AppendStash(buildFlow, phase, "end", LvlError, err.Error())
				return fmt.Errorf("build failed - aborting production")
			} else if errorCount > 0 {
				AppendStash(buildFlow, phase, "end", LvlError, "❌ build error")
			} else {
				AppendStash(buildFlow, phase, "end", LvlSuccess, "✅ success")
			}

			logDivider()
		}

		// Log termination message
		msg = fmt.Sprintf("🤖 %s %s(%v) release(flow:%s, build:%s)", summaryFlow, endPhase, statusStep, releaseFlow, releaseBuild)
		AppendStash(summaryFlow, endPhase, statusStep, LvlInfo, msg)

		// Determine final status
		if errorCount == 0 {
			msg = fmt.Sprintf("%s ✅ success, no errors", msg)
			slog.Info(msg)
			AppendStash(summaryFlow, endPhase, statusStep, LvlSuccess, msg)
			return nil
		}
		msg = fmt.Sprintf("%s ❌ %d build errors", msg, errorCount)
		slog.Error(msg)
		AppendStash(summaryFlow, endPhase, statusStep, LvlError, msg)
		return fmt.Errorf("%s - aborting production", msg)
	},
}

// flowCliParams extracts check and build steps from flags or environment variables
// Priority: flags override environment variables
func flowCliParams(cmd *cobra.Command) (checkSteps []string, buildSteps []string, reset bool) {
	// Get flag values
	checkFlag, _ := cmd.Flags().GetString("check")
	buildFlag, _ := cmd.Flags().GetString("build")
	resetFlag, _ := cmd.Flags().GetBool("reset")

	// --check flag always sets checkSteps if provided
	if checkFlag != "" {
		checkSteps = strings.Fields(checkFlag)
	} else {
		// Fall back to CHK environment variable
		checkSteps = getEnvTokens("CHK")
	}

	// --build flag always sets buildSteps if provided
	if buildFlag != "" {
		buildSteps = strings.Fields(buildFlag)
	} else {
		// Fall back to MAKE environment variable
		buildSteps = getEnvTokens("MAKE")
	}

	return checkSteps, buildSteps, resetFlag
}

// getEnvTokens splits an environment variable value by spaces
func getEnvTokens(envVar string) []string {
	value := os.Getenv(envVar)
	if value == "" {
		return []string{}
	}
	return strings.Fields(value)
}

// flowIsProduction runs "clog BC is build prod" to determine if in production mode
func flowIsProduction() bool {
	release := Releases()[0]
	return release.Build == "prod"
}

// runCheck executes a check command directly using check.Command
func runCheck(checkName string) error {
	// Execute the check command directly without spawning a shell
	args := []string{checkName}
	return check.Command.RunE(Command, args)
}

// runMakeStep executes a make step command
func runMakeStep(flow string, phase string) error {
	args := []string{"bc-" + phase, flow, phase}
	exitCode, err := scripts.Exec("clog", args, nil)
	if exitCode != 0 && err == nil {
		return fmt.Errorf("command exited with code %d", exitCode)
	}
	return err
}

// logDivider prints a log divider
func logDivider() {
	slog.Info("▬▬▬▬▬▬▬▬▬|▬▬▬▬▬▬▬▬▬|▬▬▬▬▬▬▬▬▬|▬▬▬▬▬▬▬▬▬|▬▬▬▬▬▬▬▬▬|▬▬▬▬▬▬▬▬▬|▬▬▬▬▬▬▬▬▬|▬▬▬▬▬▬▬▬▬|")
}

func init() {
	// Add flags to the flow command
	flowCmd.Flags().String("check", "", "Space-separated list of check steps to run (overrides $CHK)")
	flowCmd.Flags().String("build", "", "Space-separated list of build steps to run (overrides $MAKE)")
	flowCmd.Flags().BoolP("reset", "R", false, "reset the stash file for a new run")

	// Add flow subcommand to the main BC command
	Command.AddCommand(flowCmd)
}
