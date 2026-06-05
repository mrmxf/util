//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

package scripts_test

import (
	"testing"

	"github.com/mrmxf/util/scripts"
	. "github.com/smartystreets/goconvey/convey"
)

type TestScript struct {
	title string
	path  string
	info  scripts.ScriptInfo
}

func Test_Script_Parse(t *testing.T) {
	testfiles := []TestScript{
		{title: "3 ln     ok syntax  ok format",
			path: "../testclog/script-meta-3ln-okfmt-ok.sh",
			info: scripts.ScriptInfo{
				CmdUse:    "ClogTest3ln",
				CmdShort:  "Short help for ClogTest3ln",
				CmdLong:   "",
				NeedsOpts: false, FilePath: "",
			}},
		{title: "3 ln-ext ok syntax bad format",
			path: "../testclog/script-meta-3xln-okfmt-ok.sh",
			info: scripts.ScriptInfo{
				CmdUse:    "ClogTest3xln",
				CmdShort:  "Short Help for ClogTest3xln extra lines",
				CmdLong:   "",
				NeedsOpts: false, FilePath: "",
			}},
		{title: "2 ln     ok syntax  ok format",
			path: "../testclog/script-meta-2ln-okfmt-ok.sh",
			info: scripts.ScriptInfo{
				CmdUse:    "ClogTest2ln",
				CmdShort:  "Short help for ClogTest2ln",
				CmdLong:   "",
				NeedsOpts: false, FilePath: "",
			}},
		{title: "2 ln-ext ok syntax bad format",
			path: "../testclog/script-meta-2xln-okfmt-ok.sh",
			info: scripts.ScriptInfo{
				CmdUse:    "ClogTest2xln",
				CmdShort:  "Short help for ClogTest2xln",
				CmdLong:   "",
				NeedsOpts: false, FilePath: "",
			}},
	}

	Convey("Parse script headers correctly", t, func() {
		for _, tf := range testfiles {
			Convey("Check "+tf.title, func() {
				var info *scripts.ScriptInfo
				info, err := scripts.ParseScriptInfo(tf.path)
				So(err, ShouldBeNil)
				So(info, ShouldNotBeNil)
				So(info.CmdUse, ShouldEqual, tf.info.CmdUse)
				So(info.CmdShort, ShouldEqual, tf.info.CmdShort)
			})
		}
	})

}
