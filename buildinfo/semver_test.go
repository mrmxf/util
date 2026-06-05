//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

package buildinfo

import (
	"encoding/json"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestLinkerDataJSONParsing(t *testing.T) {
	Convey("Given the LinkerDataJSON struct", t, func() {

		Convey("When parsing a complete production build JSON", func() {
			jsonStr := `{"build":"prod","tag":"v1.2.3","hash":"587be0f365cbcd772504aefe843ecc0a51efbd46","date":"2025-08-06","suffix":"","name":"clog","title":"Command_Line_Of_Go"}`
			var data LinkerDataJSON
			err := json.Unmarshal([]byte(jsonStr), &data)

			Convey("Then it should parse without error", func() {
				So(err, ShouldBeNil)
			})

			Convey("Then all fields should match expected values", func() {
				So(data.Build, ShouldEqual, "prod")
				So(data.Tag, ShouldEqual, "v1.2.3")
				So(data.Hash, ShouldEqual, "587be0f365cbcd772504aefe843ecc0a51efbd46")
				So(data.Date, ShouldEqual, "2025-08-06")
				So(data.Suffix, ShouldEqual, "")
				So(data.AppName, ShouldEqual, "clog")
				So(data.AppTitle, ShouldEqual, "Command_Line_Of_Go")
			})
		})

		Convey("When parsing a development build JSON with suffix", func() {
			jsonStr := `{"build":"dev","tag":"v0.9.9","hash":"abc123def456789012345678901234567890abcd","date":"2025-01-15","suffix":"rc","name":"testapp","title":"Test_Application"}`
			var data LinkerDataJSON
			err := json.Unmarshal([]byte(jsonStr), &data)

			Convey("Then it should parse without error", func() {
				So(err, ShouldBeNil)
			})

			Convey("Then dev build fields should be correct", func() {
				So(data.Build, ShouldEqual, "dev")
				So(data.Suffix, ShouldEqual, "rc")
			})
		})

		Convey("When parsing the default embedded JSON constant", func() {
			var data LinkerDataJSON
			err := json.Unmarshal([]byte(LinkerDataJSONstr), &data)

			Convey("Then it should parse without error", func() {
				So(err, ShouldBeNil)
			})

			Convey("Then it should have dev build defaults", func() {
				So(data.Build, ShouldEqual, "dev")
				So(data.Hash, ShouldEqual, "")
				So(data.Date, ShouldEqual, "")
				So(data.Suffix, ShouldEqual, "")
			})
		})

		Convey("When parsing JSON with missing optional fields", func() {
			jsonStr := `{"build":"prod","tag":"v1.0.0","hash":"1234567890123456789012345678901234567890"}`
			var data LinkerDataJSON
			err := json.Unmarshal([]byte(jsonStr), &data)

			Convey("Then it should parse without error", func() {
				So(err, ShouldBeNil)
			})

			Convey("Then missing fields should be empty strings", func() {
				So(data.Date, ShouldEqual, "")
				So(data.Suffix, ShouldEqual, "")
				So(data.AppName, ShouldEqual, "")
				So(data.AppTitle, ShouldEqual, "")
			})
		})

		Convey("When parsing JSON with extra whitespace", func() {
			jsonStr := `  {"build":"prod","hash":"1234567890123456789012345678901234567890"}  `
			var data LinkerDataJSON
			err := json.Unmarshal([]byte(jsonStr), &data)

			Convey("Then it should handle whitespace gracefully", func() {
				So(err, ShouldBeNil)
				So(data.Build, ShouldEqual, "prod")
			})
		})

		Convey("When parsing malformed JSON", func() {
			jsonStr := `{"build":"prod","hash":invalid}`
			var data LinkerDataJSON
			err := json.Unmarshal([]byte(jsonStr), &data)

			Convey("Then it should return an error", func() {
				So(err, ShouldNotBeNil)
			})
		})
	})
}

func TestCleanLinkerData(t *testing.T) {
	Convey("Given the cleanLinkerData function", t, func() {

		Convey("When processing a valid production build", func() {
			originalSemVerJSON := SemVerJSON
			SemVerJSON = `{"build":"prod","tag":"v1.2.3","hash":"587be0f365cbcd772504aefe843ecc0a51efbd46","date":"2025-08-06","suffix":"","name":"clog","title":"Command_Line_Of_Go"}`

			Reset(func() {
				SemVerJSON = originalSemVerJSON
			})

			// Reset linkerData to defaults
			linkerData = &LinkerDataJSON{
				Tag:      dummyTag,
				Hash:     dummyHash,
				Date:     dummyTime,
				Suffix:   dummySuffix,
				AppName:  "",
				AppTitle: "",
			}

			err := cleanLinkerData()

			Convey("Then it should not return an error", func() {
				So(err, ShouldBeNil)
			})

			Convey("Then IsProductionBuild should be true", func() {
				So(IsProductionBuild, ShouldBeTrue)
			})

			Convey("Then the hash should be preserved", func() {
				So(linkerData.Hash, ShouldEqual, "587be0f365cbcd772504aefe843ecc0a51efbd46")
				So(len(linkerData.Hash), ShouldEqual, 40)
			})

			Convey("Then the date should be parsed correctly", func() {
				So(linkerData.Date, ShouldEqual, "2025-08-06")
			})

			Convey("Then production suffix should be empty", func() {
				So(linkerData.Suffix, ShouldEqual, "")
			})
		})

		Convey("When processing a development build", func() {
			originalSemVerJSON := SemVerJSON
			SemVerJSON = `{"build":"dev","tag":"v0.9.9","hash":"abc123def456789012345678901234567890abcd","date":"2025-01-15","suffix":"rc","name":"testapp","title":"Test_App"}`

			Reset(func() {
				SemVerJSON = originalSemVerJSON
			})

			linkerData = &LinkerDataJSON{
				Tag:      dummyTag,
				Hash:     dummyHash,
				Date:     dummyTime,
				Suffix:   dummySuffix,
				AppName:  "",
				AppTitle: "",
			}

			_ = cleanLinkerData()

			Convey("Then IsProductionBuild should be false", func() {
				So(IsProductionBuild, ShouldBeFalse)
			})

			Convey("Then dev suffix should be prepended to custom suffix", func() {
				So(linkerData.Suffix, ShouldEqual, "dev-rc")
			})
		})

		Convey("When processing data with quotes around JSON string", func() {
			originalSemVerJSON := SemVerJSON
			SemVerJSON = `"{"build":"dev","hash":"1234567890123456789012345678901234567890"}"`

			Reset(func() {
				SemVerJSON = originalSemVerJSON
			})

			linkerData = &LinkerDataJSON{
				Tag:    dummyTag,
				Hash:   dummyHash,
				Date:   dummyTime,
				Suffix: dummySuffix,
			}

			_ = cleanLinkerData()

			Convey("Then it should strip quotes and parse", func() {
				// The quotes should be stripped in the trim operation
				So(linkerData.Build, ShouldEqual, "dev")
			})
		})

		Convey("When processing data with invalid hash length", func() {
			originalSemVerJSON := SemVerJSON
			SemVerJSON = `{"build":"prod","hash":"tooshort","date":"2025-01-15"}`

			Reset(func() {
				SemVerJSON = originalSemVerJSON
			})

			linkerData = &LinkerDataJSON{
				Tag:    dummyTag,
				Hash:   "tooshort",
				Date:   dummyTime,
				Suffix: dummySuffix,
			}

			err := cleanLinkerData()

			Convey("Then it should use dummy hash", func() {
				So(err, ShouldBeNil)
				So(linkerData.Hash, ShouldEqual, dummyHash)
				So(len(linkerData.Hash), ShouldEqual, 40)
			})
		})

		Convey("When processing data with empty hash", func() {
			originalSemVerJSON := SemVerJSON
			SemVerJSON = `{"build":"prod","hash":"","date":"2025-01-15"}`

			Reset(func() {
				SemVerJSON = originalSemVerJSON
			})

			linkerData = &LinkerDataJSON{
				Tag:    dummyTag,
				Hash:   "",
				Date:   dummyTime,
				Suffix: dummySuffix,
			}

			err := cleanLinkerData()

			Convey("Then it should use dummy hash", func() {
				So(err, ShouldBeNil)
				So(linkerData.Hash, ShouldEqual, dummyHash)
			})
		})

		Convey("When processing data with invalid date format", func() {
			originalSemVerJSON := SemVerJSON
			SemVerJSON = `{"build":"prod","hash":"1234567890123456789012345678901234567890","date":"not-a-date"}`

			Reset(func() {
				SemVerJSON = originalSemVerJSON
			})

			linkerData = &LinkerDataJSON{
				Tag:    dummyTag,
				Hash:   "1234567890123456789012345678901234567890",
				Date:   "not-a-date",
				Suffix: dummySuffix,
			}

			err := cleanLinkerData()

			Convey("Then it should use current date", func() {
				So(err, ShouldBeNil)
				// Verify it's a valid date format
				_, parseErr := time.Parse("2006-01-02", linkerData.Date)
				So(parseErr, ShouldBeNil)
			})
		})

		Convey("When processing data with empty date", func() {
			originalSemVerJSON := SemVerJSON
			SemVerJSON = `{"build":"prod","hash":"1234567890123456789012345678901234567890","date":""}`

			Reset(func() {
				SemVerJSON = originalSemVerJSON
			})

			linkerData = &LinkerDataJSON{
				Tag:    dummyTag,
				Hash:   "1234567890123456789012345678901234567890",
				Date:   "",
				Suffix: dummySuffix,
			}

			err := cleanLinkerData()

			Convey("Then it should generate current date", func() {
				So(err, ShouldBeNil)
				So(linkerData.Date, ShouldNotBeEmpty)
				_, parseErr := time.Parse("2006-01-02", linkerData.Date)
				So(parseErr, ShouldBeNil)
			})
		})

		Convey("When AppName and AppTitle are empty", func() {
			originalSemVerJSON := SemVerJSON
			SemVerJSON = `{"build":"dev","hash":"1234567890123456789012345678901234567890","name":"","title":""}`

			Reset(func() {
				SemVerJSON = originalSemVerJSON
			})

			linkerData = &LinkerDataJSON{
				Tag:      dummyTag,
				Hash:     "1234567890123456789012345678901234567890",
				Date:     dummyTime,
				Suffix:   dummySuffix,
				AppName:  "",
				AppTitle: "",
			}

			err := cleanLinkerData()

			Convey("Then it should derive names from module path", func() {
				So(err, ShouldBeNil)
				// Should either get from debug.ReadBuildInfo or use defaults
				So(linkerData.AppName, ShouldNotBeEmpty)
				So(linkerData.AppTitle, ShouldNotBeEmpty)
			})
		})
	})
}

func TestVersionInfoInitialization(t *testing.T) {
	Convey("Given VersionInfo struct initialization", t, func() {

		Convey("When parsedInfo is populated with suffix", func() {
			testInfo := VersionInfo{
				AppTitle:    "Test Application",
				AppName:     "testapp",
				CommitId:    "1234567890123456789012345678901234567890",
				Tag:         "v1.0.0",
				ARCH:        "amd64",
				OS:          "linux",
				Date:        "2025-01-15",
				SuffixShort: "-rc",
				SuffixLong:  "-rc.1234",
			}

			Convey("Then Short version should include tag and short suffix", func() {
				short := testInfo.Tag + testInfo.SuffixShort
				So(short, ShouldEqual, "v1.0.0-rc")
			})

			Convey("Then Long version should include full metadata", func() {
				long := testInfo.Tag + testInfo.SuffixLong + " (" + testInfo.Date + ":" + testInfo.OS + ":" + testInfo.ARCH + ")"
				So(long, ShouldContainSubstring, "v1.0.0-rc.1234")
				So(long, ShouldContainSubstring, "2025-01-15")
				So(long, ShouldContainSubstring, "linux")
				So(long, ShouldContainSubstring, "amd64")
			})
		})

		Convey("When parsedInfo has no suffix (production)", func() {
			testInfo := VersionInfo{
				Tag:         "v2.0.0",
				SuffixShort: "",
				SuffixLong:  "",
				Date:        "2025-01-15",
				OS:          "darwin",
				ARCH:        "arm64",
			}

			Convey("Then Short version should be tag only", func() {
				short := testInfo.Tag + testInfo.SuffixShort
				So(short, ShouldEqual, "v2.0.0")
			})

			Convey("Then Long version should not have suffix", func() {
				long := testInfo.Tag + testInfo.SuffixLong + " (" + testInfo.Date + ":" + testInfo.OS + ":" + testInfo.ARCH + ")"
				So(long, ShouldStartWith, "v2.0.0 (")
				So(long, ShouldContainSubstring, "darwin:arm64")
			})
		})

		Convey("When AppTitle has underscores", func() {
			originalSemVerJSON := SemVerJSON
			SemVerJSON = `{"build":"prod","hash":"1234567890123456789012345678901234567890","title":"Command_Line_Of_Go"}`

			Reset(func() {
				SemVerJSON = originalSemVerJSON
			})

			linkerData = &LinkerDataJSON{
				Tag:      dummyTag,
				Hash:     "1234567890123456789012345678901234567890",
				Date:     dummyTime,
				Suffix:   "",
				AppTitle: "Command_Line_Of_Go",
			}

			err := cleanLinkerData()

			Convey("Then underscores should be replaced with spaces", func() {
				So(err, ShouldBeNil)
				So(parsedInfo.AppTitle, ShouldEqual, "Command Line Of Go")
			})
		})
	})
}

func TestSuffixHandling(t *testing.T) {
	Convey("Given suffix handling in cleanLinkerData", t, func() {

		Convey("When production build with no suffix", func() {
			originalSemVerJSON := SemVerJSON
			SemVerJSON = `{"build":"prod","hash":"1234567890123456789012345678901234567890","suffix":""}`

			Reset(func() {
				SemVerJSON = originalSemVerJSON
			})

			linkerData = &LinkerDataJSON{
				Build:  "prod",
				Hash:   "1234567890123456789012345678901234567890",
				Suffix: "",
			}

			err := cleanLinkerData()

			Convey("Then SuffixShort and SuffixLong should be empty", func() {
				So(err, ShouldBeNil)
				So(parsedInfo.SuffixShort, ShouldEqual, "")
				So(parsedInfo.SuffixLong, ShouldEqual, "")
			})
		})

		Convey("When production build with suffix", func() {
			originalSemVerJSON := SemVerJSON
			SemVerJSON = `{"build":"prod","hash":"abcd567890123456789012345678901234567890","suffix":"beta"}`

			Reset(func() {
				SemVerJSON = originalSemVerJSON
			})

			linkerData = &LinkerDataJSON{
				Build:  "prod",
				Hash:   "abcd567890123456789012345678901234567890",
				Suffix: "beta",
			}

			err := cleanLinkerData()

			Convey("Then suffix should be used as-is", func() {
				So(err, ShouldBeNil)
				So(parsedInfo.SuffixShort, ShouldEqual, "-beta")
				So(parsedInfo.SuffixLong, ShouldStartWith, "-beta.")
				So(parsedInfo.SuffixLong, ShouldContainSubstring, "abcd")
			})
		})

		Convey("When dev build with no suffix", func() {
			originalSemVerJSON := SemVerJSON
			SemVerJSON = `{"build":"dev","hash":"1234567890123456789012345678901234567890","suffix":""}`

			Reset(func() {
				SemVerJSON = originalSemVerJSON
			})

			linkerData = &LinkerDataJSON{
				Build:  "dev",
				Hash:   "1234567890123456789012345678901234567890",
				Suffix: "",
			}

			err := cleanLinkerData()

			Convey("Then suffix should be 'dev'", func() {
				So(err, ShouldBeNil)
				So(linkerData.Suffix, ShouldEqual, "dev")
				So(parsedInfo.SuffixShort, ShouldEqual, "-dev")
			})
		})

		Convey("When dev build with custom suffix", func() {
			originalSemVerJSON := SemVerJSON
			SemVerJSON = `{"build":"dev","hash":"1234567890123456789012345678901234567890","suffix":"rc"}`

			Reset(func() {
				SemVerJSON = originalSemVerJSON
			})

			linkerData = &LinkerDataJSON{
				Build:  "dev",
				Hash:   "1234567890123456789012345678901234567890",
				Suffix: "rc",
			}

			err := cleanLinkerData()

			Convey("Then suffix should be 'dev-' prefixed", func() {
				So(err, ShouldBeNil)
				So(linkerData.Suffix, ShouldEqual, "dev-rc")
				So(parsedInfo.SuffixShort, ShouldEqual, "-dev-rc")
			})
		})

		Convey("When suffix includes first 4 chars of commit in long form", func() {
			originalSemVerJSON := SemVerJSON
			// Hash must be exactly 40 characters (SHA-1)
			SemVerJSON = `{"build":"prod","hash":"a1b2c3d4e5f67890123456789012345678901234","suffix":"rc"}`

			Reset(func() {
				SemVerJSON = originalSemVerJSON
			})

			linkerData = &LinkerDataJSON{
				Build:  "prod",
				Hash:   "a1b2c3d4e5f67890123456789012345678901234",
				Suffix: "rc",
			}

			err := cleanLinkerData()

			Convey("Then SuffixLong should include first 4 chars of hash", func() {
				So(err, ShouldBeNil)
				So(parsedInfo.SuffixLong, ShouldEqual, "-rc.a1b2")
			})
		})
	})
}

func TestInfoFunction(t *testing.T) {
	Convey("Given the Info() function", t, func() {

		Convey("When calling Info()", func() {
			info := Info()

			Convey("Then it should return a VersionInfo struct", func() {
				So(info, ShouldHaveSameTypeAs, VersionInfo{})
			})

			Convey("Then it should have populated fields", func() {
				So(info.AppName, ShouldNotBeEmpty)
				So(info.OS, ShouldNotBeEmpty)
				So(info.ARCH, ShouldNotBeEmpty)
			})
		})
	})
}
