//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

package slogger_test

import (
	"log/slog"
	"os"
	"testing"

	"github.com/mrmxf/util/slogger"
	. "github.com/smartystreets/goconvey/convey"
)

var refLoggerType *slog.Logger

func refStyledLogger(level slog.Level) {}
func refTeeLogger(path string, level slog.Level) (*slog.Logger, *os.File, error) {
	return nil, nil, nil
}

func TestSpec(t *testing.T) {

	// Check exported elements (backwards compatibility)
	Convey("We should have consistent exported elements", t, func() {

		Convey("Exported properties", func() {
			Convey("Logger should spawn during init()", func() {
				So(slogger.Logger, ShouldNotBeNil)
			})
			Convey("Logger should be the right type", func() {
				So(slogger.Logger, ShouldHaveSameTypeAs, refLoggerType)
			})
		})

		Convey("Exported functions", func() {
			// UsePrettyLogger
			Convey("UsePrettyLogger should exist", func() {
				So(slogger.UsePrettyLogger, ShouldNotBeNil)
			})
			Convey("UsePrettyLogger should be the right type", func() {
				So(slogger.UsePrettyLogger, ShouldHaveSameTypeAs, refStyledLogger)
			})

			// UseJSONLogger
			Convey("UseJSONLogger should exist", func() {
				So(slogger.UseJSONLogger, ShouldNotBeNil)
			})
			Convey("UseJSONLogger should be the right type", func() {
				So(slogger.UseJSONLogger, ShouldHaveSameTypeAs, refStyledLogger)
			})

			// UsePlainLogger
			Convey("UsePlainLogger should exist", func() {
				So(slogger.UsePlainLogger, ShouldNotBeNil)
			})
			Convey("UsePlainLogger should be the right type", func() {
				So(slogger.UsePlainLogger, ShouldHaveSameTypeAs, refStyledLogger)
			})

			// NewTeeLogger
			Convey("NewTeeLogger should exist", func() {
				So(slogger.NewTeeLogger, ShouldNotBeNil)
			})
			Convey("NewTeeLogger should be the right type", func() {
				So(slogger.NewTeeLogger, ShouldHaveSameTypeAs, refTeeLogger)
			})
		})

	})
}
