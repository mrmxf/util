//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

package slogger

import (
	"io"
	"log/slog"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// DbgTmpLogPath is today's debug log for the current user:
// /tmp/clog-<user>-YYYY-MM-DD.log.
//
// The file is per user because it is created 0644: a single shared file would
// belong to whoever ran clog first that day, and every other user would fall
// back to console only. Opening it 0666 instead would let any local user forge
// lines in everyone's log.
func DbgTmpLogPath() string {
	return DbgTmpLogPathFor(currentUser(), time.Now())
}

// DbgTmpLogPathFor is the debug log path for username on day:
// <os.TempDir()>/clog-<username>-YYYY-MM-DD.log. Characters that are not safe
// in a file name (a Windows DOMAIN\user, say) become "_".
func DbgTmpLogPathFor(username string, day time.Time) string {
	return filepath.Join(os.TempDir(), "clog-"+safeName(username)+"-"+day.Format("2006-01-02")+".log")
}

// currentUser is the login name, or the uid when the lookup fails (a scratch
// container with no /etc/passwd entry, for example).
func currentUser() string {
	if u, err := user.Current(); err == nil && u.Username != "" {
		return u.Username
	}
	return strconv.Itoa(os.Getuid())
}

// safeName keeps letters, digits, '.', '-' and '_'; anything else becomes '_'.
func safeName(s string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '.', r == '-', r == '_':
			return r
		}
		return '_'
	}, s)
}

// NewPrettyWithDbgTmpHandler builds a MultiHandler with two sinks:
//
//   - PrettyHandler → os.Stderr at consoleLevel (coloured, human-readable)
//   - slog.JSONHandler → DbgTmpLogPath() at slog.LevelDebug (always)
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

// newDbgTmpHandler fans console out alongside the user's debug file. Every file
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
