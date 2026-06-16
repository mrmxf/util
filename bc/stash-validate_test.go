//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

package bc

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestLoadValidateStash(t *testing.T) {
	Convey("Given the stash validation functionality", t, func() {

		// Create a temporary directory for test stash files
		tempDir, err := os.MkdirTemp("", "stash-validate-test-*")
		So(err, ShouldBeNil)

		// Set up test stash path (override the bc.StashPath config hook)
		originalStashPath := StashPath
		testStashPath := filepath.Join(tempDir, "test-stash.yaml")
		StashPath = func() string { return testStashPath }

		Reset(func() {
			// Cleanup
			os.RemoveAll(tempDir)
			StashPath = originalStashPath
		})

		Convey("When validating a correctly structured stash", func() {
			// Create a valid stash
			stashData := Stash{
				Flow: map[FlowName]FlowPhases{
					"BUILD": {
						"compile": PhaseSteps{
							{Step: "go-build", Message: "compilation successful", Timestamp: time.Now()},
						},
					},
					"CHECK": {
						"lint": PhaseSteps{
							{Step: "golint", Message: "lint passed", Timestamp: time.Now()},
						},
					},
				},
			}

			err := UpdateStash(stashData)
			So(err, ShouldBeNil)

			Convey("It should validate successfully", func() {
				loaded, err := LoadValidateStash()
				So(err, ShouldBeNil)
				So(loaded.Flow, ShouldNotBeNil)
				So(len(loaded.Flow), ShouldEqual, 2)
			})
		})

		Convey("When validating stash with a malformed flow name", func() {
			// Flow/phase/step names are identifiers: must start with a letter.
			// A leading digit is rejected (case, however, is NOT constrained —
			// see TestValidateStashYAML for the mixed-case positive case).
			invalidYAML := `flow:
  9build:
    compile:
      - step: "test"
        message: "test message"
        timestamp: "2025-10-04T20:00:00Z"
`
			err := os.WriteFile(testStashPath, []byte(invalidYAML), 0644)
			So(err, ShouldBeNil)

			Convey("It should return validation error", func() {
				_, err := LoadValidateStash()
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "validation failed")
			})
		})

		Convey("When validating stash with a malformed phase name", func() {
			// Phase name with a space is not a valid identifier.
			invalidYAML := `flow:
  BUILD:
    "bad phase":
      - step: "test"
        message: "test message"
        timestamp: "2025-10-04T20:00:00Z"
`
			err := os.WriteFile(testStashPath, []byte(invalidYAML), 0644)
			So(err, ShouldBeNil)

			Convey("It should return validation error", func() {
				_, err := LoadValidateStash()
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "validation failed")
			})
		})

		Convey("When validating stash with extra fields", func() {
			// Note: LoadValidateStash loads via Go struct which automatically drops unknown fields
			// Use ValidateStashYAML for strict validation before loading
			invalidYAML := `flow:
  BUILD:
    compile:
      - step: "test"
        message: "test message"
        timestamp: "2025-10-04T20:00:00Z"
        extra_field: "should not be here"
`
			Convey("ValidateStashYAML should catch extra fields", func() {
				err := ValidateStashYAML([]byte(invalidYAML))
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "validation failed")
			})

			Convey("LoadValidateStash will pass because struct unmarshaling drops extra fields", func() {
				err := os.WriteFile(testStashPath, []byte(invalidYAML), 0644)
				So(err, ShouldBeNil)

				// This passes because Go struct acts as a filter, dropping unknown fields
				_, err = LoadValidateStash()
				So(err, ShouldBeNil)
			})
		})

		Convey("When validating stash with missing required fields", func() {
			// Create stash missing message field
			invalidYAML := `flow:
  BUILD:
    compile:
      - step: "test"
        timestamp: "2025-10-04T20:00:00Z"
`
			err := os.WriteFile(testStashPath, []byte(invalidYAML), 0644)
			So(err, ShouldBeNil)

			Convey("It should return validation error", func() {
				_, err := LoadValidateStash()
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "validation failed")
			})
		})
	})
}

func TestValidateStashYAML(t *testing.T) {
	Convey("Given the ValidateStashYAML function", t, func() {

		Convey("When validating valid YAML", func() {
			validYAML := `flow:
  BUILD:
    compile:
      - step: "go-build"
        message: "compilation successful"
        timestamp: "2025-10-04T20:00:00Z"
`
			err := ValidateStashYAML([]byte(validYAML))

			Convey("It should validate successfully", func() {
				So(err, ShouldBeNil)
			})
		})

		Convey("When validating YAML with a malformed flow name", func() {
			// Leading digit → not a valid identifier → rejected.
			invalidYAML := `flow:
  9build:
    compile:
      - step: "test"
        message: "test"
        timestamp: "2025-10-04T20:00:00Z"
`
			err := ValidateStashYAML([]byte(invalidYAML))

			Convey("It should return validation error", func() {
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "validation failed")
			})
		})

		Convey("When validating mixed-case names (the real convention)", func() {
			// Letter-case is intentionally NOT constrained: the uppercase
			// summary flow "FLOW" coexists with lowercase flows like "build",
			// and SlackStash must accept any real stash. This must validate.
			okYAML := `flow:
  FLOW:
    check:
      - step: "start"
        message: "begin"
        timestamp: "2025-10-04T20:00:00Z"
  build:
    compile:
      - step: "go-build"
        message: "ok"
        timestamp: "2025-10-04T20:00:01Z"
`
			err := ValidateStashYAML([]byte(okYAML))

			Convey("It should validate successfully", func() {
				So(err, ShouldBeNil)
			})
		})

		Convey("When validating malformed YAML", func() {
			malformedYAML := `this is not valid yaml: [{`

			err := ValidateStashYAML([]byte(malformedYAML))

			Convey("It should return parse error", func() {
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "failed to parse YAML")
			})
		})
	})
}
