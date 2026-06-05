//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

//
// package cmdlog adds a Log command to the clog command line tool
// without polluting the slogger package with cobra dependency baggage

package cmdlog

import (
	"fmt"
	"runtime"
	"strings"

	slog "github.com/mrmxf/util/slogger"

	"github.com/spf13/cobra"
)

var debug bool
var emergency bool
var error bool
var fatal bool
var info bool
var success bool
var trace bool
var warn bool
var up bool

// var Command is the cobra definition of the Log command
var Command = &cobra.Command{
	Use:     "Log",
	Short:   helpShort,
	Long:    helpLong,
	Example: helpExample,
	Run: func(cmd *cobra.Command, args []string) {
		logMsg := strings.Join(args, " ")
		// most serious flag wins
		logFlag := "none"

		if up {
			//up one line, start of line, del EOL
			fmt.Print("\x1b[A\x1b[G\x1b[K")
		}
		// if user has many flags, then the top-most case statement wins
		switch {
		case emergency:
			slog.Emergency(logMsg)
			logFlag = "X"
		case fatal:
			slog.Fatal(logMsg)
			logFlag = "F"
		case error:
			slog.Error(logMsg)
			logFlag = "E"
		case warn:
			slog.Warn(logMsg)
			logFlag = "W"
		case success:
			slog.Success(logMsg)
			logFlag = "S"
		case info:
			slog.Info(logMsg)
			logFlag = "I"
		case trace:
			slog.Trace(logMsg)
			logFlag = "T"
		case debug:
			slog.Debug(logMsg)
			logFlag = "D"
		}
		// level, levelFile := slogger.GetLogLevel()
		slog.Debug("Log (-%s) (%s)", logFlag, logMsg)
	},
}

func init() {
	_, file, _, _ := runtime.Caller(0)
	slog.Debug("init " + file)

	Command.PersistentFlags().BoolVarP(&info, "info", "I", false, "clog Log -I \"Info message\"")
	Command.PersistentFlags().BoolVarP(&success, "success", "S", false, "clog Log -S \"Success message\"")
	Command.PersistentFlags().BoolVarP(&warn, "warn", "W", false, "clog Log -W \"Warn message\"")
	Command.PersistentFlags().BoolVarP(&error, "error", "E", false, "clog Log -E \"Error message\"")
	Command.PersistentFlags().BoolVarP(&trace, "trace", "T", false, "clog Log -T \"Trace message\"")
	Command.PersistentFlags().BoolVarP(&debug, "debug", "D", false, "clog Log -D \"Debug message\"")
	Command.PersistentFlags().BoolVarP(&fatal, "fatal", "F", false, "clog Log -E \"Fatal message\"")
	Command.PersistentFlags().BoolVarP(&emergency, "emergency", "X", false, "clog Log -X \"Emergency message\"")
	Command.PersistentFlags().BoolVarP(&up, "up", "U", false, "clog Log -UI \"up (overprint) Info message\"")
}
