//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

package buildinfo

import (
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

// TestParseLinkerJSON_PureFunction tests the pure function version
// These tests don't mutate global state and can run in parallel
func TestParseLinkerJSON_PureFunction(t *testing.T) {
	Convey("Given the ParseLinkerJSON pure function", t, func() {

		Convey("When parsing a complete production build JSON", func() {
			jsonStr := `{"build":"prod","tag":"v1.2.3","hash":"587be0f365cbcd772504aefe843ecc0a51efbd46","date":"2025-08-06","suffix":"","name":"clog","title":"Command_Line_Of_Go"}`

			info, isProd, err := ParseLinkerJSON(jsonStr)

			Convey("Then it should not return an error", func() {
				So(err, ShouldBeNil)
			})

			Convey("Then isProd should be true", func() {
				So(isProd, ShouldBeTrue)
			})

			Convey("Then the hash should be preserved", func() {
				So(info.CommitId, ShouldEqual, "587be0f365cbcd772504aefe843ecc0a51efbd46")
				So(len(info.CommitId), ShouldEqual, 40)
			})

			Convey("Then the date should be parsed correctly", func() {
				So(info.Date, ShouldEqual, "2025-08-06")
			})

			Convey("Then AppTitle underscores should be replaced with spaces", func() {
				So(info.AppTitle, ShouldEqual, "Command Line Of Go")
			})

			Convey("Then Short version should be correct", func() {
				So(info.Short, ShouldEqual, "v1.2.3")
			})

			Convey("Then Long version should include metadata", func() {
				So(info.Long, ShouldContainSubstring, "v1.2.3")
				So(info.Long, ShouldContainSubstring, "2025-08-06")
			})
		})

		Convey("When parsing a development build with suffix", func() {
			jsonStr := `{"build":"dev","tag":"v0.9.9","hash":"abc1234567890123456789012345678901234567","date":"2025-01-15","suffix":"rc","name":"testapp","title":"Test_App"}`

			info, isProd, err := ParseLinkerJSON(jsonStr)

			Convey("Then it should not return an error", func() {
				So(err, ShouldBeNil)
			})

			Convey("Then isProd should be false", func() {
				So(isProd, ShouldBeFalse)
			})

			Convey("Then dev suffix should be prepended", func() {
				So(info.SuffixShort, ShouldEqual, "-dev-rc")
			})

			Convey("Then SuffixLong should include commit hash prefix", func() {
				So(info.SuffixLong, ShouldEqual, "-dev-rc.abc1")
			})

			Convey("Then Short version should include suffix", func() {
				So(info.Short, ShouldEqual, "v0.9.9-dev-rc")
			})
		})

		Convey("When parsing JSON with quotes around it", func() {
			jsonStr := `"{"build":"dev","hash":"1234567890123456789012345678901234567890"}"`

			info, _, err := ParseLinkerJSON(jsonStr)

			Convey("Then it should strip quotes and parse", func() {
				So(err, ShouldBeNil)
				So(info.CommitId, ShouldEqual, "1234567890123456789012345678901234567890")
			})
		})

		Convey("When parsing JSON with invalid hash length", func() {
			jsonStr := `{"build":"prod","hash":"tooshort","date":"2025-01-15"}`

			info, _, err := ParseLinkerJSON(jsonStr)

			Convey("Then it should use dummy hash", func() {
				So(err, ShouldBeNil)
				So(info.CommitId, ShouldEqual, dummyHash)
				So(len(info.CommitId), ShouldEqual, 40)
			})
		})

		Convey("When parsing JSON with empty hash", func() {
			jsonStr := `{"build":"prod","hash":"","date":"2025-01-15"}`

			info, _, err := ParseLinkerJSON(jsonStr)

			Convey("Then it should use dummy hash", func() {
				So(err, ShouldBeNil)
				So(info.CommitId, ShouldEqual, dummyHash)
			})
		})

		Convey("When parsing JSON with invalid date format", func() {
			jsonStr := `{"build":"prod","hash":"1234567890123456789012345678901234567890","date":"not-a-date"}`

			info, _, err := ParseLinkerJSON(jsonStr)

			Convey("Then it should use current date", func() {
				So(err, ShouldBeNil)
				// Verify it's a valid date format
				_, parseErr := time.Parse("2006-01-02", info.Date)
				So(parseErr, ShouldBeNil)
			})
		})

		Convey("When parsing JSON with empty date", func() {
			jsonStr := `{"build":"prod","hash":"1234567890123456789012345678901234567890","date":""}`

			info, _, err := ParseLinkerJSON(jsonStr)

			Convey("Then it should generate current date", func() {
				So(err, ShouldBeNil)
				So(info.Date, ShouldNotBeEmpty)
				_, parseErr := time.Parse("2006-01-02", info.Date)
				So(parseErr, ShouldBeNil)
			})
		})

		Convey("When AppName and AppTitle are empty", func() {
			jsonStr := `{"build":"dev","hash":"1234567890123456789012345678901234567890","name":"","title":""}`

			info, _, err := ParseLinkerJSON(jsonStr)

			Convey("Then it should derive names from module path", func() {
				So(err, ShouldBeNil)
				So(info.AppName, ShouldNotBeEmpty)
				So(info.AppTitle, ShouldNotBeEmpty)
			})
		})

		Convey("When parsing malformed JSON", func() {
			jsonStr := `{"build":"prod","hash":invalid}`

			_, _, err := ParseLinkerJSON(jsonStr)

			Convey("Then it should return an error", func() {
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "failed to parse semver JSON")
			})
		})

		Convey("When production build has no suffix", func() {
			jsonStr := `{"build":"prod","hash":"1234567890123456789012345678901234567890","suffix":""}`

			info, isProd, err := ParseLinkerJSON(jsonStr)

			Convey("Then suffixes should be empty", func() {
				So(err, ShouldBeNil)
				So(isProd, ShouldBeTrue)
				So(info.SuffixShort, ShouldEqual, "")
				So(info.SuffixLong, ShouldEqual, "")
			})
		})

		Convey("When production build has suffix", func() {
			jsonStr := `{"build":"prod","hash":"abcd567890123456789012345678901234567890","suffix":"beta"}`

			info, isProd, err := ParseLinkerJSON(jsonStr)

			Convey("Then suffix should be used as-is (not prefixed with dev)", func() {
				So(err, ShouldBeNil)
				So(isProd, ShouldBeTrue)
				So(info.SuffixShort, ShouldEqual, "-beta")
				So(info.SuffixLong, ShouldStartWith, "-beta.")
				So(info.SuffixLong, ShouldContainSubstring, "abcd")
			})
		})

		Convey("When dev build with no suffix", func() {
			jsonStr := `{"build":"dev","hash":"1234567890123456789012345678901234567890","suffix":""}`

			info, isProd, err := ParseLinkerJSON(jsonStr)

			Convey("Then suffix should be 'dev'", func() {
				So(err, ShouldBeNil)
				So(isProd, ShouldBeFalse)
				So(info.SuffixShort, ShouldEqual, "-dev")
			})
		})

		Convey("When dev build with custom suffix", func() {
			jsonStr := `{"build":"dev","hash":"1234567890123456789012345678901234567890","suffix":"rc"}`

			info, isProd, err := ParseLinkerJSON(jsonStr)

			Convey("Then suffix should be 'dev-' prefixed", func() {
				So(err, ShouldBeNil)
				So(isProd, ShouldBeFalse)
				So(info.SuffixShort, ShouldEqual, "-dev-rc")
			})
		})

		Convey("When suffix includes first 4 chars of commit in long form", func() {
			jsonStr := `{"build":"prod","hash":"a1b2c3d4e5f67890123456789012345678901234","suffix":"rc"}`

			info, _, err := ParseLinkerJSON(jsonStr)

			Convey("Then SuffixLong should include first 4 chars of hash", func() {
				So(err, ShouldBeNil)
				So(info.SuffixLong, ShouldEqual, "-rc.a1b2")
			})
		})
	})
}

// TestParseLinkerJSON_Concurrent demonstrates that pure functions can be tested in parallel
func TestParseLinkerJSON_Concurrent(t *testing.T) {
	t.Parallel() // This test can run in parallel with others

	Convey("Given concurrent ParseLinkerJSON calls", t, func() {
		Convey("When multiple goroutines parse different JSONs", func() {
			// Parse prod build
			info1, isProd1, err1 := ParseLinkerJSON(`{"build":"prod","tag":"v1.0.0","hash":"1234567890123456789012345678901234567890"}`)

			// Parse dev build
			info2, isProd2, err2 := ParseLinkerJSON(`{"build":"dev","tag":"v2.0.0","hash":"abcd567890123456789012345678901234567890"}`)

			Convey("Then both should succeed independently", func() {
				So(err1, ShouldBeNil)
				So(err2, ShouldBeNil)

				So(isProd1, ShouldBeTrue)
				So(isProd2, ShouldBeFalse)

				So(info1.Tag, ShouldEqual, "v1.0.0")
				So(info2.Tag, ShouldEqual, "v2.0.0")

				So(info1.Short, ShouldEqual, "v1.0.0")
				So(info2.Short, ShouldEqual, "v2.0.0-dev")
			})
		})
	})
}

// TestParseLinkerJSON_IsolatedState demonstrates test isolation
func TestParseLinkerJSON_IsolatedState(t *testing.T) {
	Convey("Given isolated ParseLinkerJSON calls", t, func() {

		Convey("Test 1: Parse prod build", func() {
			info, isProd, _ := ParseLinkerJSON(`{"build":"prod","tag":"v1.0.0","hash":"1234567890123456789012345678901234567890"}`)

			So(isProd, ShouldBeTrue)
			So(info.Short, ShouldEqual, "v1.0.0")
		})

		Convey("Test 2: Parse dev build (independent of Test 1)", func() {
			info, isProd, _ := ParseLinkerJSON(`{"build":"dev","tag":"v2.0.0","hash":"abcd567890123456789012345678901234567890"}`)

			// This test is completely isolated from Test 1
			// No need to reset any state
			So(isProd, ShouldBeFalse)
			So(info.Short, ShouldEqual, "v2.0.0-dev")
		})

		Convey("Test 3: Parse another prod build (still independent)", func() {
			info, isProd, _ := ParseLinkerJSON(`{"build":"prod","tag":"v3.0.0","hash":"9999567890123456789012345678901234567890"}`)

			// No state pollution from previous tests
			So(isProd, ShouldBeTrue)
			So(info.Short, ShouldEqual, "v3.0.0")
		})
	})
}
