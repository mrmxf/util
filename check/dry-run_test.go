//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

package check_test

import (
	"testing"

	"github.com/mrmxf/util/check"
	. "github.com/smartystreets/goconvey/convey"
)

// helpers to build the raw block arrays that kfg.Raw.Get would return

func block(pairs ...any) map[string]interface{} {
	m := make(map[string]interface{}, len(pairs)/2)
	for i := 0; i+1 < len(pairs); i += 2 {
		m[pairs[i].(string)] = pairs[i+1]
	}
	return m
}

func blocks(blks ...map[string]interface{}) []any {
	out := make([]any, len(blks))
	for i, b := range blks {
		out[i] = b
	}
	return out
}

func TestDryRun(t *testing.T) {
	Convey("DryRun", t, func() {

		Convey("nil input returns one error describing the nil", func() {
			errs := check.DryRun("grp", nil)
			So(len(errs), ShouldEqual, 1)
			So(errs[0].Field, ShouldBeEmpty)
		})

		Convey("non-array input returns one error", func() {
			errs := check.DryRun("grp", "not-an-array")
			So(len(errs), ShouldEqual, 1)
		})

		Convey("empty array returns no errors", func() {
			errs := check.DryRun("grp", []any{})
			So(errs, ShouldBeEmpty)
		})

		Convey("valid block with all known string fields returns no errors", func() {
			raw := blocks(block(
				"name", "check go",
				"try", "go version",
				"ok", "echo ok",
				"catch", "echo fail",
				"finally", "echo done",
			))
			errs := check.DryRun("tools", raw)
			So(errs, ShouldBeEmpty)
		})

		Convey("unrecognised field is reported", func() {
			raw := blocks(block("try", "go version", "bogus", "val"))
			errs := check.DryRun("tools", raw)
			So(len(errs), ShouldEqual, 1)
			So(errs[0].Field, ShouldEqual, "bogus")
			So(errs[0].Block, ShouldEqual, 1)
			So(errs[0].Section, ShouldEqual, "tools")
		})

		Convey("'try' as a list of conditions is valid (new schema)", func() {
			raw := blocks(block("try", []string{"go version", "git --version"}))
			errs := check.DryRun("tools", raw)
			So(errs, ShouldBeEmpty)
		})

		Convey("wrong type for 'try' field is reported", func() {
			raw := blocks(block("try", 99))
			errs := check.DryRun("tools", raw)
			So(len(errs), ShouldBeGreaterThanOrEqualTo, 1)
			found := false
			for _, e := range errs {
				if e.Field == "try" {
					found = true
				}
			}
			So(found, ShouldBeTrue)
		})

		Convey("wrong type for 'ok' field is reported", func() {
			raw := blocks(block("try", "go version", "ok", 42))
			errs := check.DryRun("tools", raw)
			So(len(errs), ShouldBeGreaterThanOrEqualTo, 1)
			found := false
			for _, e := range errs {
				if e.Field == "ok" {
					found = true
				}
			}
			So(found, ShouldBeTrue)
		})

		Convey("block that is a scalar (not a map) is reported", func() {
			raw := []any{"not-a-map"}
			errs := check.DryRun("grp", raw)
			So(len(errs), ShouldEqual, 1)
			So(errs[0].Block, ShouldEqual, 1)
		})

		Convey("errors accumulate across multiple bad blocks", func() {
			raw := blocks(
				block("bogus1", "x"),                      // unrecognised field
				block("try", 99),                          // wrong type
				block("try", "ok", "also-bogus", "y"),     // unrecognised field
			)
			errs := check.DryRun("sec", raw)
			So(len(errs), ShouldBeGreaterThan, 1)
		})

		Convey("named block uses name in ID field", func() {
			raw := blocks(block("name", "my check", "bogus", "x"))
			errs := check.DryRun("sec", raw)
			So(len(errs), ShouldEqual, 1)
			So(errs[0].ID, ShouldContainSubstring, "my check")
		})

		Convey("ParseError.Error() includes section, id, and field", func() {
			e := check.ParseError{Section: "tools", Block: 2, ID: "block #2", Field: "bogus", Reason: "unrecognised field"}
			So(e.Error(), ShouldContainSubstring, "tools")
			So(e.Error(), ShouldContainSubstring, "bogus")
		})

		Convey("ParseError.Error() omits field name when Field is empty", func() {
			e := check.ParseError{Section: "tools", Block: 1, ID: "block #1", Field: "", Reason: "some structural error"}
			msg := e.Error()
			So(msg, ShouldContainSubstring, "tools")
			So(msg, ShouldContainSubstring, "some structural error")
			// field name must not appear as a dotted suffix
			So(msg, ShouldNotContainSubstring, " .")
		})
	})
}

func TestValidBlockFields(t *testing.T) {
	Convey("ValidBlockFields contains the expected keys", t, func() {
		for _, f := range []string{"name", "try", "ok", "catch", "finally", "before"} {
			So(check.ValidBlockFields[f], ShouldBeTrue)
		}
	})
}
