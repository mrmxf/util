//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

package check_test

import (
	"bytes"
	"testing"

	"github.com/mrmxf/util/check"
	. "github.com/smartystreets/goconvey/convey"
)

func TestFormatGroups(t *testing.T) {

	Convey("FormatGroups with no groups", t, func() {
		var buf bytes.Buffer
		n := check.FormatGroups(&buf, nil)

		Convey("returns zero", func() {
			So(n, ShouldEqual, 0)
		})
		Convey("still prints the header", func() {
			So(buf.String(), ShouldContainSubstring, "check groups in config key `check`")
		})
		Convey("tells the user how to fix it", func() {
			So(buf.String(), ShouldContainSubstring, "none found")
		})
	})

	Convey("FormatGroups with groups", t, func() {
		var buf bytes.Buffer
		n := check.FormatGroups(&buf, []check.GroupInfo{
			{Name: "golang", BlockCount: 1},
			{Name: "hugo", BlockCount: 4},
			{Name: "pre-build", BlockCount: 11},
			{Name: "broken", BlockCount: -1},
		})
		out := buf.String()

		Convey("returns the group count", func() {
			So(n, ShouldEqual, 4)
		})
		Convey("prints a copy-pastable command per group", func() {
			So(out, ShouldContainSubstring, "clog Check golang")
			So(out, ShouldContainSubstring, "clog Check hugo")
			So(out, ShouldContainSubstring, "clog Check pre-build")
		})
		Convey("pads names to the longest so counts line up", func() {
			So(out, ShouldContainSubstring, "clog Check golang     1 block")
			So(out, ShouldContainSubstring, "clog Check pre-build  11 blocks")
		})
		Convey("pluralises the block count", func() {
			So(out, ShouldContainSubstring, "1 block\n")
			So(out, ShouldContainSubstring, "4 blocks")
		})
		Convey("flags a group with no usable blocks key", func() {
			So(out, ShouldContainSubstring, "no `blocks:` key")
		})
	})
}
