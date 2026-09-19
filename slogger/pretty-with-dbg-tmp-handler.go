//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

package slogger

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

// DbgTmpLogPath is today's shared debug log: /tmp/clog-YYYY-MM-DD.log.
func DbgTmpLogPath() string {
	return filepath.Join(os.TempDir(), "clog-"+time.Now().Format("2006-01-02")+".log")
}

// NewPrettyWithDbgTmpHandler builds a MultiHandler with two sinks:
//
//   - PrettyHandler → os.Stderr at consoleLevel (coloured, human-readable)
//   - slog.JSONHandler → /tmp/clog-YYYY-MM-DD.log at slog.LevelDebug (always)
//
// The returned io.Closer wraps the log file; call Close() on app exit.
// If the file cannot be opened the function falls back to the console handler
// only, logs a warning, and returns the error alongside a no-op Closer.
func NewPrettyWithDbgTmpHandler(consoleLevel slog.Level) (slog.Handler, io.Closer, error) {
	return newDbgTmpHandler(NewPrettyHandler(os.Stderr, &PrettyHandlerOptions{Level: consoleLevel}))
}

// NewCiWithDbgTmpHandler is NewPrettyWithDbgTmpHandler for CI consoles: the
// console line has no timestamp (the runner stamps every line already), the
// debug file keeps full timestamps for correlating with the runner's log.
func NewCiWithDbgTmpHandler(consoleLevel slog.Level) (slog.Handler, io.Closer, error) {
	return newDbgTmpHandler(NewPrettyHandler(os.Stderr, &PrettyHandlerOptions{Level: consoleLevel, OmitTime: true}))
}

// newDbgTmpHandler fans console out alongside the shared debug file. Every file
// record carries pid: one snippet runs clog many times, and all of them append
// to the same file.
func newDbgTmpHandler(console slog.Handler) (slog.Handler, io.Closer, error) {
	logPath := DbgTmpLogPath()
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		slog.New(console).Warn("debug log unavailable, console only", "path", logPath, "err", err)
		return console, io.NopCloser(nil), err
	}

	fileHandler := slog.NewJSONHandler(f, &slog.HandlerOptions{Level: slog.LevelDebug}).
		WithAttrs([]slog.Attr{slog.Int("pid", os.Getpid())})
	return NewMultiHandler(console, fileHandler), f, nil
}
