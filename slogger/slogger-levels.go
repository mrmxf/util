//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

package slogger

import (
	"context"
	"log/slog"
)

const (
	LevelTrace     = slog.Level(-8)
	LevelDebug     = slog.LevelDebug
	LevelInfo      = slog.LevelInfo
	LevelSuccess   = slog.Level(2)
	LevelWarn      = slog.LevelWarn
	LevelError     = slog.LevelError
	LevelFatal     = slog.Level(10)
	LevelEmergency = slog.Level(12)
)

const strTrace string = "TRACE"
const strDebug string = "DEBUG"
const strInfo string = "INFO"
const strSuccess string = "SUCCESS"
const strWarn string = "WARN"
const strError string = "ERROR"
const strFatal string = "FATAL"
const strEmergency string = "EMERGENCY"

// sloggerReplaceAttr is a function that injects custom level names into slog
func sloggerReplaceAttr(groups []string, a slog.Attr) slog.Attr {
	// Remove time from the output for predictable test output.
	// if a.Key == slog.TimeKey {
	// 	return slog.Attr{}
	// }

	// Customize the name of the level key and the output string, including
	// custom level values.
	if a.Key == slog.LevelKey {
		// Optionally rename the level key from "level" to "banana" or something else.
		a.Key = "level"

		// Handle custom level values.
		level := a.Value.Any().(slog.Level)
		// This could also look up the name from a map or other structure, but
		// this demonstrates using a switch statement to rename levels. For
		// maximum performance, the string values should be constants, but this
		// example uses the raw strings for readability.
		switch {
		case level >= LevelEmergency:
			a.Value = slog.StringValue(strEmergency)
		case level >= LevelFatal:
			a.Value = slog.StringValue(strFatal)
		case level >= LevelError:
			a.Value = slog.StringValue(strError)
		case level >= LevelWarn:
			a.Value = slog.StringValue(strWarn)
		case level >= LevelSuccess:
			a.Value = slog.StringValue(strSuccess)
		case level >= LevelInfo:
			a.Value = slog.StringValue(strInfo)
		case level >= LevelDebug:
			a.Value = slog.StringValue(strDebug)
		default:
			a.Value = slog.StringValue(strTrace)
		}
	}
	return a
}

// Default returns the default [Logger].
func Default() *slog.Logger { return slog.Default() }

// Debug logs at [LevelDebug].
func Debug(msg string, args ...any) {
	Default().Log(context.Background(), LevelDebug, msg, args...)
}

// DebugContext logs at [LevelDebug] with the given context.
func DebugContext(ctx context.Context, msg string, args ...any) {
	Default().Log(ctx, LevelDebug, msg, args...)
}

// Trace logs at [LevelTrace].
func Trace(msg string, args ...any) {
	Default().Log(context.Background(), LevelTrace, msg, args...)
}

// TraceContext logs at [LevelTrace] with the given context.
func TraceContext(ctx context.Context, msg string, args ...any) {
	Default().Log(ctx, LevelTrace, msg, args...)
}

// Info logs at [LevelInfo].
func Info(msg string, args ...any) {
	Default().Log(context.Background(), LevelInfo, msg, args...)
}

// InfoContext logs at [LevelInfo] with the given context.
func InfoContext(ctx context.Context, msg string, args ...any) {
	Default().Log(ctx, LevelInfo, msg, args...)
}

// Success logs at [LevelSuccess].
func Success(msg string, args ...any) {
	Default().Log(context.Background(), LevelSuccess, msg, args...)
}

// SuccessContext logs at [LevelSuccess] with the given context.
func SuccessContext(ctx context.Context, msg string, args ...any) {
	Default().Log(ctx, LevelSuccess, msg, args...)
}

// Warn logs at [LevelWarn].
func Warn(msg string, args ...any) {
	Default().Log(context.Background(), LevelWarn, msg, args...)
}

// WarnContext logs at [LevelWarn] with the given context.
func WarnContext(ctx context.Context, msg string, args ...any) {
	Default().Log(ctx, LevelWarn, msg, args...)
}

// Error logs at [LevelError].
func Error(msg string, args ...any) {
	Default().Log(context.Background(), LevelError, msg, args...)
}

// ErrorContext logs at [LevelError] with the given context.
func ErrorContext(ctx context.Context, msg string, args ...any) {
	Default().Log(ctx, LevelError, msg, args...)
}

// Fatal logs at [LevelFatal].
func Fatal(msg string, args ...any) {
	Default().Log(context.Background(), LevelFatal, msg, args...)
}

// FatalContext logs at [LevelFatal] with the given context.
func FatalContext(ctx context.Context, msg string, args ...any) {
	Default().Log(ctx, LevelFatal, msg, args...)
}

// Emergency logs at [LevelEmergency].
func Emergency(msg string, args ...any) {
	Default().Log(context.Background(), LevelEmergency, msg, args...)
}

// EmergencyContext logs at [LevelEmergency] with the given context.
func EmergencyContext(ctx context.Context, msg string, args ...any) {
	Default().Log(ctx, LevelEmergency, msg, args...)
}
