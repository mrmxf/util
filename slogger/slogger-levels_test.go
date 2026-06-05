//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

package slogger_test

import (
	"bufio"
	"bytes"
	"fmt"
	"strings"
	"testing"
	"time"

	slog "github.com/mrmxf/util/slogger"
	. "github.com/smartystreets/goconvey/convey"
)

func stripEscapes(s string) string {
	inEscSeq := false
	var res string = ""

	for _, r := range s {
		if inEscSeq {
			if r == 'm' {
				inEscSeq = false
			}
		} else {
			if r == '\x1b' {
				inEscSeq = true
			} else {
				res += string(r)
			}
		}
	}
	return res
}

func checkLogOutput(t *testing.T, buf bytes.Buffer, title string, expected string) {
	ltt := len("2006-01-02 15:04:05")
	raw := buf.String()
	// remove any escape sequences from pretty printing
	got := stripEscapes(raw)
	msg := strings.TrimSpace(got)
	ttt := strings.TrimSpace(got)
	if len(got) > ltt {
		msg = strings.TrimSpace(got[ltt:])
		ttt = strings.TrimSpace(got[:ltt])
	}
	// all pretty print prefixes are 3 chars except "OK" which is " OK" in the main code
	// to prevent a false failure, pad the message with " " if it's an OK message
	if strings.HasPrefix(msg, "OK") {
		msg = " " + msg
	}
	Convey(fmt.Sprintf("%s check", title), func() {

		Convey("bufio len count", func() {
			So(len(got), ShouldEqual, ltt+len(expected)+2)
		})
		Convey(fmt.Sprintf("output string == \"%s\"", expected), func() {
			So(msg, ShouldEqual, expected)
		})
		logTimestamp, err := time.Parse("2006-01-02 15:04:05", ttt)
		Convey(fmt.Sprintf("time stamp == \"%s\"", logTimestamp.Format("2006-01-02 15:04:05")), func() {
			So(err, ShouldBeNil)
			// the function call should be no more than 1 second in the past
			callTimeLimit := time.Until(logTimestamp).Milliseconds()
			So(callTimeLimit, ShouldBeGreaterThan, -1000)
		})

	})
}

func TestSpec_Levels(t *testing.T) {
	// Check each level works correctly
	Convey("Each level should work properly", t, func() {
		buf := bytes.NewBuffer(nil)
		out := bufio.NewWriter(buf)
		slog.UsePrettyIoLogger(out, slog.LevelTrace)

		buf.Reset()
		slog.Trace("Trace")
		out.Flush()
		checkLogOutput(t, *buf, "Trace", "TRC Trace")

		buf.Reset()
		slog.Debug("Debug")
		out.Flush()
		checkLogOutput(t, *buf, "Debug", "DBG Debug")

		buf.Reset()
		slog.Info("Info")
		out.Flush()
		checkLogOutput(t, *buf, "Info", "INF Info")

		buf.Reset()
		slog.Success("Success")
		out.Flush()
		checkLogOutput(t, *buf, "Success", " OK Success")

		buf.Reset()
		slog.Warn("Warn")
		out.Flush()
		checkLogOutput(t, *buf, "Warn", "WRN Warn")

		buf.Reset()
		slog.Error("Error")
		out.Flush()
		checkLogOutput(t, *buf, "Error", "ERR Error")

		buf.Reset()
		slog.Fatal("Fatal")
		out.Flush()
		checkLogOutput(t, *buf, "Fatal", "FTL Fatal")

		buf.Reset()
		slog.Emergency("Emergency")
		out.Flush()
		checkLogOutput(t, *buf, "Emergency", "!!! Emergency")

	})

}
