//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

package slogger

// package slogger defines a consistent set of styled loggers for clog and
// other apps. It silently initializes to a Pretty Logger with LogInfo logging.
//
// If you want a different default logger then use a different UseXXXLogger
// in your main.init()

import (
    "io"
    "log/slog"
    "runtime"
)

type Slogger struct {
    *slog.Logger
}

// the exported default logger
var Logger *slog.Logger

// these will be exported with a get function for clog Log
var logLevel slog.Level
var logLevelFile slog.Level

type SlogStyle int

// create an enum for the different types of logger
const (
    StylePlain SlogStyle = iota
    StylePretty
    StyleJSON
    StyleTee
    StyleJob
    StyleNats
    StylePrettyWithDbgTmp // PrettyHandler to stderr + JSONHandler to /tmp/clog-YYYY-MM-DD.log
)

// add a string function to Sprintf("%s") our new type
func (s SlogStyle) String() string {
    switch s {
    case StylePlain:
        return "  plain"
    case StylePretty:
        return " pretty"
    case StyleJSON:
        return "   JSON"
    case StyleJob:
        return "    job"
    case StyleTee:
        return "    tee"
    case StyleNats:
        return "   nats"
    case StylePrettyWithDbgTmp:
        return "dbgtmp"
    }
    return "unknown"
}

// the defaults
// var defaultLogLevel = slog.LevelDebug  //use this for init tracing
var defaultLogLevel = slog.LevelInfo
var defaultLogStyle = StylePrettyWithDbgTmp

// logCloser holds the Closer for the active logger's file handle (if any).
// Call CloseLogger() on app exit to flush and close it.
var logCloser io.Closer

// CloseLogger closes any open file handle used by the current logger (e.g. the
// debug tmp log). Safe to call when no file is open. Typically deferred in main:
//
//	defer slogger.CloseLogger()
func CloseLogger() error {
    if logCloser != nil {
        return logCloser.Close()
    }
    return nil
}

// use this function to set a log level from a config file
func SetLogger(level slog.Level, style SlogStyle) {
    switch style {
    case StylePlain:
        UsePlainLogger(level)
    case StylePretty:
        UsePrettyLogger(level)
    case StyleJSON:
        UseJSONLogger(level)
    case StyleJob:
        UseJobLogger(level)
    case StylePrettyWithDbgTmp:
        UsePrettyWithDbgTmpLogger(level)
    default:
        // there is no default Tee logger as it needs a file path
        SetLogger(level, StylePretty)
    }
}

// get the active log levels for a split console / cicd logging experience
func GetLogLevel() (logLevel slog.Level, logLevelFile slog.Level) {
    return logLevel, logLevelFile
}

func init() {
    // uncomment this line to see init order
    SetLogger(defaultLogLevel, defaultLogStyle)

    // trace init order for sanity
    _, file, _, _ := runtime.Caller(0)
    slog.Debug("init " + file)
}
