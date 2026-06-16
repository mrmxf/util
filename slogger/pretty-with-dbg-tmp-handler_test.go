//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

package slogger_test

import (
	"bufio"
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mrmxf/util/slogger"
	. "github.com/smartystreets/goconvey/convey"
)

func TestPrettyWithDbgTmpHandler(t *testing.T) {
	Convey("NewPrettyWithDbgTmpHandler", t, func() {

		Convey("returns a non-nil handler and closer without error", func() {
			h, closer, err := slogger.NewPrettyWithDbgTmpHandler(slog.LevelInfo)
			So(err, ShouldBeNil)
			So(h, ShouldNotBeNil)
			So(closer, ShouldNotBeNil)
			closer.Close()
		})

		Convey("creates the date-suffixed log file in os.TempDir()", func() {
			expectedPath := filepath.Join(os.TempDir(), "clog-"+time.Now().Format("2006-01-02")+".log")

			_, closer, err := slogger.NewPrettyWithDbgTmpHandler(slog.LevelInfo)
			So(err, ShouldBeNil)
			defer closer.Close()

			_, statErr := os.Stat(expectedPath)
			So(statErr, ShouldBeNil)
		})

		Convey("a DEBUG record appears in the file even when console level is INFO", func() {
			logPath := filepath.Join(os.TempDir(), "clog-"+time.Now().Format("2006-01-02")+".log")

			h, closer, err := slogger.NewPrettyWithDbgTmpHandler(slog.LevelInfo)
			So(err, ShouldBeNil)
			defer closer.Close()

			// get file size before writing
			statBefore, _ := os.Stat(logPath)
			sizeBefore := int64(0)
			if statBefore != nil {
				sizeBefore = statBefore.Size()
			}

			// write a DEBUG record — below console level but must reach the file
			ctx := context.Background()
			rec := slog.NewRecord(time.Now(), slog.LevelDebug, "dbg-probe-message", 0)
			So(h.Handle(ctx, rec), ShouldBeNil)

			// flush: the file handler is unbuffered (JSONHandler writes directly)
			// re-open and scan from sizeBefore
			f, openErr := os.Open(logPath)
			So(openErr, ShouldBeNil)
			defer f.Close()

			_, _ = f.Seek(sizeBefore, 0)
			found := false
			scanner := bufio.NewScanner(f)
			for scanner.Scan() {
				line := scanner.Text()
				if strings.Contains(line, "dbg-probe-message") {
					var m map[string]any
					if json.Unmarshal([]byte(line), &m) == nil {
						found = true
					}
				}
			}
			So(found, ShouldBeTrue)
		})

		Convey("handler is Enabled for DEBUG level even when console level is INFO", func() {
			h, closer, err := slogger.NewPrettyWithDbgTmpHandler(slog.LevelInfo)
			So(err, ShouldBeNil)
			defer closer.Close()

			// MultiHandler.Enabled returns true if ANY child accepts the level.
			// The file handler runs at Debug, so Debug records must be accepted.
			So(h.Enabled(context.Background(), slog.LevelDebug), ShouldBeTrue)
			So(h.Enabled(context.Background(), slog.LevelInfo), ShouldBeTrue)
		})
	})
}

func TestUsePrettyWithDbgTmpLogger(t *testing.T) {
	Convey("UsePrettyWithDbgTmpLogger", t, func() {

		Convey("sets the global Logger and returns a closer", func() {
			closer, err := slogger.UsePrettyWithDbgTmpLogger(slog.LevelInfo)
			So(err, ShouldBeNil)
			So(closer, ShouldNotBeNil)
			So(slogger.Logger, ShouldNotBeNil)
			closer.Close()
		})

		Convey("CloseLogger() is safe to call after UsePrettyWithDbgTmpLogger", func() {
			_, _ = slogger.UsePrettyWithDbgTmpLogger(slog.LevelInfo)
			So(slogger.CloseLogger(), ShouldBeNil)
		})
	})
}
