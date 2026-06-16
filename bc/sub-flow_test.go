//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

package bc

import (
	"os"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestFlowCommand(t *testing.T) {
	Convey("Given the BC flow command", t, func() {

		Convey("When checking command structure", func() {
			Convey("It should have proper metadata", func() {
				So(flowCmd.Use, ShouldEqual, "flow")
				So(flowCmd.Short, ShouldNotBeEmpty)
				So(flowCmd.Long, ShouldNotBeEmpty)
			})

			Convey("It should describe environment variables", func() {
				So(flowCmd.Long, ShouldContainSubstring, "CHK")
				So(flowCmd.Long, ShouldContainSubstring, "MAKE")
			})

			Convey("It should document exit codes", func() {
				So(flowCmd.Long, ShouldContainSubstring, "Exit codes")
			})
		})

		Convey("When checking helper functions", func() {
			Convey("getEnvTokens should split space-separated values", func() {
				// Save and restore environment
				originalValue := ""
				if val, exists := LookupEnv("TEST_TOKENS"); exists {
					originalValue = val
				}
				Reset(func() {
					if originalValue != "" {
						Setenv("TEST_TOKENS", originalValue)
					} else {
						Unsetenv("TEST_TOKENS")
					}
				})

				Setenv("TEST_TOKENS", "token1 token2 token3")
				tokens := getEnvTokens("TEST_TOKENS")

				So(len(tokens), ShouldEqual, 3)
				So(tokens[0], ShouldEqual, "token1")
				So(tokens[1], ShouldEqual, "token2")
				So(tokens[2], ShouldEqual, "token3")
			})

			Convey("getEnvTokens should handle empty values", func() {
				tokens := getEnvTokens("NONEXISTENT_VAR")
				So(len(tokens), ShouldEqual, 0)
			})
		})

		Convey("When flow command is registered", func() {
			Convey("It should be a subcommand of BC", func() {
				found := false
				for _, cmd := range Command.Commands() {
					if cmd.Name() == "flow" {
						found = true
						break
					}
				}
				So(found, ShouldBeTrue)
			})

			Convey("It should have check, build, and reset flags", func() {
				checkFlag := flowCmd.Flags().Lookup("check")
				buildFlag := flowCmd.Flags().Lookup("build")
				resetFlag := flowCmd.Flags().Lookup("reset")

				So(checkFlag, ShouldNotBeNil)
				So(buildFlag, ShouldNotBeNil)
				So(resetFlag, ShouldNotBeNil)
				So(checkFlag.Usage, ShouldContainSubstring, "check steps")
				So(buildFlag.Usage, ShouldContainSubstring, "build steps")
				So(resetFlag.Usage, ShouldContainSubstring, "reset")
			})
		})

		Convey("When using flowCliParams", func() {
			// Save original environment
			originalChk := os.Getenv("CHK")
			originalMake := os.Getenv("MAKE")
			Reset(func() {
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

			Convey("With flags set, it should use flag values", func() {
				os.Setenv("CHK", "env-check1 env-check2")
				os.Setenv("MAKE", "env-make1 env-make2")

				flowCmd.Flags().Set("check", "flag-check1 flag-check2")
				flowCmd.Flags().Set("build", "flag-build1 flag-build2")

				checkSteps, buildSteps, _ := flowCliParams(flowCmd)

				So(len(checkSteps), ShouldEqual, 2)
				So(checkSteps[0], ShouldEqual, "flag-check1")
				So(checkSteps[1], ShouldEqual, "flag-check2")
				So(len(buildSteps), ShouldEqual, 2)
				So(buildSteps[0], ShouldEqual, "flag-build1")
				So(buildSteps[1], ShouldEqual, "flag-build2")

				// Reset flags
				flowCmd.Flags().Set("check", "")
				flowCmd.Flags().Set("build", "")
			})

			Convey("With no flags, it should use environment variables", func() {
				os.Setenv("CHK", "env-check1 env-check2 env-check3")
				os.Setenv("MAKE", "env-make1 env-make2")

				flowCmd.Flags().Set("check", "")
				flowCmd.Flags().Set("build", "")

				checkSteps, buildSteps, _ := flowCliParams(flowCmd)

				So(len(checkSteps), ShouldEqual, 3)
				So(checkSteps[0], ShouldEqual, "env-check1")
				So(checkSteps[1], ShouldEqual, "env-check2")
				So(checkSteps[2], ShouldEqual, "env-check3")
				So(len(buildSteps), ShouldEqual, 2)
				So(buildSteps[0], ShouldEqual, "env-make1")
				So(buildSteps[1], ShouldEqual, "env-make2")
			})

			Convey("With only check flag set, it should use check flag and MAKE env", func() {
				os.Setenv("CHK", "env-check1")
				os.Setenv("MAKE", "env-make1 env-make2")

				flowCmd.Flags().Set("check", "flag-check1")
				flowCmd.Flags().Set("build", "")

				checkSteps, buildSteps, _ := flowCliParams(flowCmd)

				So(len(checkSteps), ShouldEqual, 1)
				So(checkSteps[0], ShouldEqual, "flag-check1")
				So(len(buildSteps), ShouldEqual, 2)
				So(buildSteps[0], ShouldEqual, "env-make1")
				So(buildSteps[1], ShouldEqual, "env-make2")

				// Reset flags
				flowCmd.Flags().Set("check", "")
			})

			Convey("With only build flag set, it should use build flag and CHK env", func() {
				os.Setenv("CHK", "env-check1 env-check2")
				os.Setenv("MAKE", "env-make1")

				flowCmd.Flags().Set("check", "")
				flowCmd.Flags().Set("build", "flag-build1")

				checkSteps, buildSteps, _ := flowCliParams(flowCmd)

				So(len(checkSteps), ShouldEqual, 2)
				So(checkSteps[0], ShouldEqual, "env-check1")
				So(checkSteps[1], ShouldEqual, "env-check2")
				So(len(buildSteps), ShouldEqual, 1)
				So(buildSteps[0], ShouldEqual, "flag-build1")

				// Reset flags
				flowCmd.Flags().Set("build", "")
			})

			Convey("With no flags and no env vars, it should return empty slices", func() {
				os.Unsetenv("CHK")
				os.Unsetenv("MAKE")

				flowCmd.Flags().Set("check", "")
				flowCmd.Flags().Set("build", "")

				checkSteps, buildSteps, _ := flowCliParams(flowCmd)

				So(len(checkSteps), ShouldEqual, 0)
				So(len(buildSteps), ShouldEqual, 0)
			})

			Convey("With reset flag set, it should return true for reset", func() {
				flowCmd.Flags().Set("reset", "true")

				_, _, reset := flowCliParams(flowCmd)

				So(reset, ShouldBeTrue)

				// Reset flag
				flowCmd.Flags().Set("reset", "false")
			})

			Convey("With reset flag not set, it should return false for reset", func() {
				flowCmd.Flags().Set("reset", "false")

				_, _, reset := flowCliParams(flowCmd)

				So(reset, ShouldBeFalse)
			})
		})
	})
}

// Helper functions for environment variable manipulation in tests
func Setenv(key, value string) {
	_ = os.Setenv(key, value)
}

func Unsetenv(key string) {
	_ = os.Unsetenv(key)
}

func LookupEnv(key string) (string, bool) {
	return os.LookupEnv(key)
}
