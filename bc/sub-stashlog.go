//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

package bc

import (
	"fmt"
	"runtime"
	"strings"

	slog "github.com/mrmxf/util/slogger"

	"github.com/spf13/cobra"
)

var slDebug bool
var slEmergency bool
var slError bool
var slFatal bool
var slInfo bool
var slSuccess bool
var slTrace bool
var slWarn bool
var slUp bool
var slFlow string
var slPhase string
var slStep string

// stashLogCmd logs a message like Log command and also adds it to the stash
var stashLogCmd = &cobra.Command{
	Use:           "stashLog",
	SilenceErrors: true,
	SilenceUsage:  true,
	Short:         "log a message to the configured logger and add to stash",
	Long:          stashLogLongHelp,
	Example:       stashLogExample,
	Run: func(cmd *cobra.Command, args []string) {
		logMsg := strings.Join(args, " ")
		// most serious flag wins
		logFlag := "none"
		var stashMsg string
		var stashLevel StashLevels

		if slUp {
			//up one line, start of line, del EOL
			fmt.Print("\x1b[A\x1b[G\x1b[K")
		}

		// if user has many flags, then the top-most case statement wins
		switch {
		case slEmergency:
			slog.Emergency(logMsg)
			logFlag = "X"
			stashMsg = "❌ EMERGENCY: " + logMsg
			stashLevel = LvlEmergency
		case slFatal:
			slog.Fatal(logMsg)
			logFlag = "F"
			stashMsg = "💀 FATAL: " + logMsg
			stashLevel = LvlFatal
		case slError:
			slog.Error(logMsg)
			logFlag = "E"
			stashMsg = "❌ ERROR: " + logMsg
			stashLevel = LvlError
		case slWarn:
			slog.Warn(logMsg)
			logFlag = "W"
			stashMsg = "⚠️  WARN: " + logMsg
			stashLevel = LvlWarn
		case slSuccess:
			slog.Success(logMsg)
			logFlag = "S"
			stashMsg = "✅ SUCCESS: " + logMsg
			stashLevel = LvlSuccess
		case slInfo:
			slog.Info(logMsg)
			logFlag = "I"
			stashMsg = "ℹ️  INFO: " + logMsg
			stashLevel = LvlInfo
		case slTrace:
			slog.Trace(logMsg)
			logFlag = "T"
			stashMsg = "🔍 TRACE: " + logMsg
			stashLevel = LvlTrace
		case slDebug:
			slog.Debug(logMsg)
			logFlag = "D"
			stashMsg = "🐛 DEBUG: " + logMsg
			stashLevel = LvlDebug
		default:
			// No log level flag specified - just use the message as-is
			logFlag = "I"
			stashMsg = logMsg
			stashLevel = LvlInfo
		}

		slog.Debug("StashLog (-%s) flow:%s phase:%s step:%s (%s)", logFlag, slFlow, slPhase, slStep, logMsg)

		// Add to stash if flow and phase are specified
		if slFlow != "" && slPhase != "" {
			// Use step if provided, otherwise use a default
			step := slStep
			if step == "" {
				step = "log"
			}

			if err := AppendStash(slFlow, slPhase, step, stashLevel, stashMsg); err != nil {
				slog.Error("failed to append to stash", "error", err)
			}
		} else if slFlow != "" || slPhase != "" {
			slog.Warn("stashLog requires both --flow (-1) and --phase (-2) to write to stash (nothing written stash)")
		}
	},
}

func init() {
	_, file, _, _ := runtime.Caller(0)
	slog.Debug("init " + file)

	// Log level flags
	stashLogCmd.PersistentFlags().BoolVarP(&slInfo, "info", "I", false, "clog BC stashLog -I \"Info message\"")
	stashLogCmd.PersistentFlags().BoolVarP(&slSuccess, "success", "S", false, "clog BC stashLog -S \"Success message\"")
	stashLogCmd.PersistentFlags().BoolVarP(&slWarn, "warn", "W", false, "clog BC stashLog -W \"Warn message\"")
	stashLogCmd.PersistentFlags().BoolVarP(&slError, "error", "E", false, "clog BC stashLog -E \"Error message\"")
	stashLogCmd.PersistentFlags().BoolVarP(&slTrace, "trace", "T", false, "clog BC stashLog -T \"Trace message\"")
	stashLogCmd.PersistentFlags().BoolVarP(&slDebug, "debug", "D", false, "clog BC stashLog -D \"Debug message\"")
	stashLogCmd.PersistentFlags().BoolVarP(&slFatal, "fatal", "F", false, "clog BC stashLog -E \"Fatal message\"")
	stashLogCmd.PersistentFlags().BoolVarP(&slEmergency, "emergency", "X", false, "clog BC stashLog -X \"Emergency message\"")
	stashLogCmd.PersistentFlags().BoolVarP(&slUp, "up", "U", false, "clog BC stashLog -UI \"up (overprint) Info message\"")

	// Stash-specific flags
	stashLogCmd.PersistentFlags().StringVarP(&slFlow, "flow", "1", "", "flow name for stash organization")
	stashLogCmd.PersistentFlags().StringVarP(&slPhase, "phase", "2", "", "phase name for stash organization")
	stashLogCmd.PersistentFlags().StringVarP(&slStep, "step", "3", "", "step name (optional, prepended to message)")

	// Add stashLog subcommand to the main BC command
	Command.AddCommand(stashLogCmd)
}
