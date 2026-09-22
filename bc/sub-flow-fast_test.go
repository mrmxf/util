//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/

package bc

import (
	"os"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

// --fast drops the WHOLE check list. Not a shorter list, not the cheap checks
// only - nothing. `clog build` checked everything, `clog build --fast` checked
// nothing; two sentences, no footnotes.
func TestFlowFastDropsEveryCheck(t *testing.T) {
	Convey("Given --fast", t, func() {
		originalChk := os.Getenv("CHK")
		originalMake := os.Getenv("MAKE")
		Reset(func() {
			flowCmd.Flags().Set("fast", "false")
			flowCmd.Flags().Set("check", "")
			flowCmd.Flags().Set("build", "")
			if originalChk != "" {
				os.Setenv("CHK", originalChk)
			} else {
				os.Unsetenv("CHK")
			}
			if originalMake != "" {
				os.Setenv("MAKE", originalMake)
			} else {
				os.Unsetenv("MAKE")
			}
		})

		Convey("it empties the check list even when $CHK names phases", func() {
			os.Setenv("CHK", "pre-build lint scan")
			os.Setenv("MAKE", "hugo ko")
			flowCmd.Flags().Set("check", "")
			flowCmd.Flags().Set("fast", "true")

			checkSteps, buildSteps, _ := flowCliParams(flowCmd)

			So(len(checkSteps), ShouldEqual, 0)
			// The build still happens: --fast is "do the minimum quickly",
			// not "do nothing".
			So(len(buildSteps), ShouldEqual, 2)
		})

		Convey("it beats an explicit --check too", func() {
			os.Unsetenv("CHK")
			flowCmd.Flags().Set("check", "lint")
			flowCmd.Flags().Set("fast", "true")

			checkSteps, _, _ := flowCliParams(flowCmd)

			So(len(checkSteps), ShouldEqual, 0)
		})

		Convey("without it the check list survives", func() {
			os.Setenv("CHK", "pre-build lint")
			flowCmd.Flags().Set("check", "")
			flowCmd.Flags().Set("fast", "false")

			checkSteps, _, _ := flowCliParams(flowCmd)

			So(len(checkSteps), ShouldEqual, 2)
		})
	})
}
