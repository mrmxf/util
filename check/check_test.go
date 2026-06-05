//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

//

package check_test

import (
	"testing"

	"github.com/mrmxf/util/check"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/spf13/cobra"
)

var refCommand *cobra.Command = nil
var refCommandRunE func(*cobra.Command, []string) error = nil

func TestSpec(t *testing.T) {

	// Check exported elements (backwards compatibility)
	Convey("We should have consistent exported elements", t, func() {

		Convey("Exported properties", func() {
			//Command
			Convey("Command should exist", func() {
				So(check.Command, ShouldNotBeNil)
			})
			Convey("Command should be the right type", func() {
				So(check.Command, ShouldHaveSameTypeAs, refCommand)
			})
			//Command.Use
			Convey("Command.Use should exist", func() {
				So(check.Command.Use, ShouldNotBeNil)
			})
			Convey("Command.Use should be \"Check\"", func() {
				So(check.Command.Use, ShouldEqual, "Check")
			})
			//YamlKey
			Convey("YamlKey should exist", func() {
				So(check.KfgKey, ShouldNotBeNil)
			})
			Convey("YamlKey should be \"heck\"", func() {
				So(check.KfgKey, ShouldEqual, "check")
			})

		})

		Convey("Exported functions", func() {
			// Command.RunE (returns error)
			Convey("Command.RunE should exist", func() {
				So(check.Command.RunE, ShouldNotBeNil)
			})
			Convey("Command.RunE should be the right type", func() {
				So(check.Command.RunE, ShouldHaveSameTypeAs, refCommandRunE)
			})

		})

	})
	// --------------------------------------------------------------------------
	// Check yaml parsing
	Convey("Parser should whine about bad yaml", t, func() {

		// Convey("check blocks", func() {
		// 	Convey("Logger should spawn during init()", func() {
		// 		So(check.Logger, ShouldNotBeNil)
		// 	})
		// 	Convey("Logger should be the right type", func() {
		// 		So(check.Logger, ShouldHaveSameTypeAs, refLoggerType)
		// 	})
		// })

	})
}
