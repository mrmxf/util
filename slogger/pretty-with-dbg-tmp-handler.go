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

// NewPrettyWithDbgTmpHandler builds a MultiHandler with two sinks:
//
//   - PrettyHandler → os.Stderr at consoleLevel (coloured, human-readable)
//   - slog.JSONHandler → /tmp/clog-YYYY-MM-DD.log at slog.LevelDebug (always)
//
// The returned io.Closer wraps the log file; call Close() on app exit.
// If the file cannot be opened the function falls back to the console handler
// only, logs a warning, and returns the error alongside a no-op Closer.
func NewPrettyWithDbgTmpHandler(consoleLevel slog.Level) (slog.Handler, io.Closer, error) {
	consoleHandler := NewPrettyHandler(os.Stderr, &PrettyHandlerOptions{Level: consoleLevel})

	logPath := filepath.Join(os.TempDir(), "clog-"+time.Now().Format("2006-01-02")+".log")
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		slog.Warn("debug log unavailable, console only", "path", logPath, "err", err)
		return consoleHandler, io.NopCloser(nil), err
	}

	fileHandler := slog.NewJSONHandler(f, &slog.HandlerOptions{Level: slog.LevelDebug})
	return NewMultiHandler(consoleHandler, fileHandler), f, nil
}
