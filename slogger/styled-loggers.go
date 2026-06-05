//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

package slogger

// package log defines the logger for the app

import (
    "bufio"
    "io"
    "log/slog"
    "os"
    "runtime"
)

func UsePrettyLogger(level slog.Level) {
    Logger = slog.New(
        NewPrettyHandler(os.Stderr, &PrettyHandlerOptions{Level: level}))
    slog.SetDefault(Logger)
    logLevel = level
}

func UsePrettyNatsLogger(natsUrl, appName string, level slog.Level) {
    prettyHandler := NewPrettyHandler(os.Stderr, &PrettyHandlerOptions{Level: level})
    
    natsHandler, err := NewNATSHandler(&NATSHandlerOptions{
        NATSUrl:       natsUrl,
        SubjectBase:   "logs",
        AppName:       appName,
        ParentHandler: prettyHandler,
    })
    
    if err != nil || natsHandler == nil {
        Logger = slog.New(prettyHandler)
        slog.SetDefault(Logger)
        logLevel = level
        return
    }
    
    Logger = slog.New(natsHandler)
    slog.SetDefault(Logger)
    logLevel = level
}

func UsePrettyIoLogger(out io.Writer, level slog.Level) {
    Logger = slog.New(
        NewPrettyHandler(out, &PrettyHandlerOptions{Level: level}))
    slog.SetDefault(Logger)
    logLevel = level
}

func UsePlainLogger(level slog.Level) {
    Logger = slog.New(
        NewPrettyHandler(os.Stderr,
            &PrettyHandlerOptions{Level: level, NoColor: true}))
    slog.SetDefault(Logger)
    logLevel = level
}

// TeeLogger is a no-color version of the PrettyLogger that is created
// to append to a job log folder. If the file cannot be opened for appending
// an error is returned
func NewTeeLogger(path string, level slog.Level) (*slog.Logger, *os.File, error) {
    fileHandle, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
    writer := bufio.NewWriter(fileHandle)

    newLogger := slog.New(
        NewPrettyHandler(writer,
            &PrettyHandlerOptions{Level: level, NoColor: true}))

    logLevelFile = level
    return newLogger, fileHandle, err
}

func UseJSONLogger(level slog.Level) {
    Logger = slog.New(slog.NewJSONHandler(os.Stderr,
        &slog.HandlerOptions{Level: level}))
    slog.SetDefault(Logger)
    logLevel = level
}

// Job logger is currently just a JSON Logger
// @ToDo - implement the full SMPTE ST 2126 logging
func UseJobLogger(level slog.Level) {
    Logger = slog.New(slog.NewJSONHandler(os.Stderr,
        &slog.HandlerOptions{Level: level}))
    slog.SetDefault(Logger)
    logLevel = level
}

func init() {
    // trace init order for sanity
    _, file, _, _ := runtime.Caller(0)
    slog.Debug("init " + file)
}
