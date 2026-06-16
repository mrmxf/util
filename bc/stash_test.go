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

func TestStashFunctions(t *testing.T) {
	Convey("Given the stash functionality", t, func() {

		// Create a temporary directory for test stash files
		tempDir, err := os.MkdirTemp("", "stash-test-*")
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

		Convey("When loading a non-existent stash file", func() {
			stashData := LoadStash()

			Convey("It should return an empty Stash with initialized Flow map", func() {
				So(stashData.Flow, ShouldNotBeNil)
				So(len(stashData.Flow), ShouldEqual, 0)
			})
		})

		Convey("When updating stash with new data", func() {
			stashData := Stash{
				Flow: map[FlowName]FlowPhases{
					"build": {
						"compile": PhaseSteps{
							{
								Step:      "go-build",
								Message:   "compilation successful",
								Timestamp: time.Now(),
							},
						},
					},
				},
			}

			err := UpdateStash(stashData)

			Convey("It should write the file without error", func() {
				So(err, ShouldBeNil)
			})

			Convey("The file should exist", func() {
				_, err := os.Stat(testStashPath)
				So(err, ShouldBeNil)
			})

			Convey("The data should be retrievable", func() {
				loaded := LoadStash()
				So(len(loaded.Flow), ShouldEqual, 1)
				So(loaded.Flow["build"], ShouldNotBeNil)
				So(len(loaded.Flow["build"]["compile"]), ShouldEqual, 1)
				So(loaded.Flow["build"]["compile"][0].Step, ShouldEqual, "go-build")
				So(loaded.Flow["build"]["compile"][0].Message, ShouldEqual, "compilation successful")
			})
		})

		Convey("When appending to an empty stash", func() {
			err := AppendStash("deploy", "prepare", "start", LvlInfo, "deployment started")

			Convey("It should create the flow, phase and add the step", func() {
				So(err, ShouldBeNil)

				loaded := LoadStash()
				So(len(loaded.Flow), ShouldEqual, 1)
				So(loaded.Flow["deploy"], ShouldNotBeNil)
				So(loaded.Flow["deploy"]["prepare"], ShouldNotBeNil)
				So(len(loaded.Flow["deploy"]["prepare"]), ShouldEqual, 1)
				So(loaded.Flow["deploy"]["prepare"][0].Step, ShouldEqual, "start")
				So(loaded.Flow["deploy"]["prepare"][0].Message, ShouldEqual, "deployment started")
			})
		})

		Convey("When appending to an existing flow and phase", func() {
			// First append
			err := AppendStash("release", "build", "compile", LvlSuccess, "build completed")
			So(err, ShouldBeNil)

			// Second append to same phase
			err = AppendStash("release", "build", "test", LvlSuccess, "tests passed")
			So(err, ShouldBeNil)

			Convey("It should preserve previous steps and add new ones", func() {
				loaded := LoadStash()
				So(len(loaded.Flow), ShouldEqual, 1)
				So(loaded.Flow["release"], ShouldNotBeNil)
				So(loaded.Flow["release"]["build"], ShouldNotBeNil)
				So(len(loaded.Flow["release"]["build"]), ShouldEqual, 2)
				So(loaded.Flow["release"]["build"][0].Step, ShouldEqual, "compile")
				So(loaded.Flow["release"]["build"][0].Message, ShouldEqual, "build completed")
				So(loaded.Flow["release"]["build"][1].Step, ShouldEqual, "test")
				So(loaded.Flow["release"]["build"][1].Message, ShouldEqual, "tests passed")
			})
		})

		Convey("When appending to multiple flows and phases", func() {
			err := AppendStash("flow1", "phase1", "step1", LvlInfo, "message1")
			So(err, ShouldBeNil)

			err = AppendStash("flow2", "phase2", "step2", LvlTrace, "message2")
			So(err, ShouldBeNil)

			err = AppendStash("flow1", "phase2", "step3", LvlWarn, "message3")
			So(err, ShouldBeNil)

			Convey("It should maintain separate flows and phases", func() {
				loaded := LoadStash()
				So(len(loaded.Flow), ShouldEqual, 2)

				So(loaded.Flow["flow1"], ShouldNotBeNil)
				So(len(loaded.Flow["flow1"]), ShouldEqual, 2)
				So(len(loaded.Flow["flow1"]["phase1"]), ShouldEqual, 1)
				So(loaded.Flow["flow1"]["phase1"][0].Step, ShouldEqual, "step1")
				So(loaded.Flow["flow1"]["phase1"][0].Message, ShouldEqual, "message1")
				So(len(loaded.Flow["flow1"]["phase2"]), ShouldEqual, 1)
				So(loaded.Flow["flow1"]["phase2"][0].Step, ShouldEqual, "step3")
				So(loaded.Flow["flow1"]["phase2"][0].Message, ShouldEqual, "message3")

				So(loaded.Flow["flow2"], ShouldNotBeNil)
				So(len(loaded.Flow["flow2"]), ShouldEqual, 1)
				So(len(loaded.Flow["flow2"]["phase2"]), ShouldEqual, 1)
				So(loaded.Flow["flow2"]["phase2"][0].Step, ShouldEqual, "step2")
				So(loaded.Flow["flow2"]["phase2"][0].Message, ShouldEqual, "message2")
			})
		})

		Convey("When steps have timestamps", func() {
			before := time.Now()
			err := AppendStash("timed", "test", "step1", LvlInfo, "first")
			So(err, ShouldBeNil)

			time.Sleep(10 * time.Millisecond)

			err = AppendStash("timed", "test", "step2", LvlInfo, "second")
			So(err, ShouldBeNil)
			after := time.Now()

			Convey("Timestamps should be in chronological order", func() {
				loaded := LoadStash()
				So(len(loaded.Flow["timed"]["test"]), ShouldEqual, 2)

				ts1 := loaded.Flow["timed"]["test"][0].Timestamp
				ts2 := loaded.Flow["timed"]["test"][1].Timestamp

				So(ts1.Before(ts2), ShouldBeTrue)
				So(ts1.After(before), ShouldBeTrue)
				So(ts2.Before(after), ShouldBeTrue)
			})
		})

		Convey("When updating with complex data", func() {
			stashData := Stash{
				Flow: map[FlowName]FlowPhases{
					"build": {
						"check": PhaseSteps{
							{Step: "lint", Message: "lint passed", Timestamp: time.Now()},
							{Step: "test", Message: "all tests passed", Timestamp: time.Now()},
						},
						"make": PhaseSteps{
							{Step: "compile", Message: "compilation successful", Timestamp: time.Now()},
							{Step: "package", Message: "artifacts created", Timestamp: time.Now()},
						},
					},
					"deploy": {
						"staging": PhaseSteps{
							{Step: "upload", Message: "deployed to staging", Timestamp: time.Now()},
						},
					},
				},
			}

			err := UpdateStash(stashData)

			Convey("It should preserve all flows, phases and steps", func() {
				So(err, ShouldBeNil)

				loaded := LoadStash()
				So(len(loaded.Flow), ShouldEqual, 2)
				So(len(loaded.Flow["build"]), ShouldEqual, 2)
				So(len(loaded.Flow["build"]["check"]), ShouldEqual, 2)
				So(len(loaded.Flow["build"]["make"]), ShouldEqual, 2)
				So(len(loaded.Flow["deploy"]), ShouldEqual, 1)
				So(len(loaded.Flow["deploy"]["staging"]), ShouldEqual, 1)
			})
		})

		Convey("When the stash path is empty", func() {
			StashPath = func() string { return "" }

			// The compiled default is tmp/ClogBcStash.yaml. (This assertion
			// previously expected the historical ".action-stash" name and had
			// been failing in clog-mrmxf; corrected here to the actual default.)
			Convey("It should use the default stash path", func() {
				path := getStashPath()
				absPath, _ := filepath.Abs("tmp/ClogBcStash.yaml")
				So(path, ShouldEqual, absPath)
			})
		})
	})
}

func TestStashTypes(t *testing.T) {
	Convey("Given the Stash types", t, func() {

		Convey("FlowPhaseStep should have required fields", func() {
			step := FlowPhaseStep{
				Step:      "test-step",
				Message:   "test message",
				Timestamp: time.Now(),
			}

			So(step.Step, ShouldEqual, "test-step")
			So(step.Message, ShouldEqual, "test message")
			So(step.Timestamp, ShouldHaveSameTypeAs, time.Time{})
		})

		Convey("PhaseSteps should be a slice of FlowPhaseStep", func() {
			steps := PhaseSteps{
				{Step: "step1", Message: "msg1", Timestamp: time.Now()},
				{Step: "step2", Message: "msg2", Timestamp: time.Now()},
			}

			So(len(steps), ShouldEqual, 2)
		})

		Convey("FlowPhases should be a map of PhaseName to PhaseSteps", func() {
			phases := FlowPhases{
				"build": PhaseSteps{
					{Step: "compile", Message: "built", Timestamp: time.Now()},
				},
				"test": PhaseSteps{
					{Step: "unit", Message: "passed", Timestamp: time.Now()},
				},
			}

			So(len(phases), ShouldEqual, 2)
			So(phases["build"], ShouldNotBeNil)
			So(phases["test"], ShouldNotBeNil)
		})

		Convey("Stash should have Flow map", func() {
			stash := Stash{
				Flow: map[FlowName]FlowPhases{
					"release": {
						"build": PhaseSteps{
							{Step: "step1", Message: "msg1", Timestamp: time.Now()},
						},
					},
				},
			}

			So(stash.Flow, ShouldNotBeNil)
			So(len(stash.Flow), ShouldEqual, 1)
			So(stash.Flow["release"], ShouldNotBeNil)
		})
	})
}
