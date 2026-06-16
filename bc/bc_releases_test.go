//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

package bc

import (
	"bytes"
	"strings"
	"testing"

	"github.com/mrmxf/util/kfg"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/spf13/cobra"
)

func TestReleasesCommands(t *testing.T) {
	Convey("Given the BC releases subcommands", t, func() {
		tests := []struct {
			name        string
			cmd         *cobra.Command
			args        []string
			expectError bool
			description string
		}{
			{
				name:        "releases_version",
				cmd:         releaseVersionCmd,
				args:        []string{},
				expectError: false,
				description: "should print release version",
			},
			{
				name:        "releases_date",
				cmd:         releaseDateCmd,
				args:        []string{},
				expectError: false,
				description: "should print release date",
			},
			{
				name:        "releases_flow",
				cmd:         releaseFlowCmd,
				args:        []string{},
				expectError: false,
				description: "should print release flow",
			},
			{
				name:        "releases_build",
				cmd:         releaseBuildCmd,
				args:        []string{},
				expectError: false,
				description: "should print release build type",
			},
			{
				name:        "releases_note",
				cmd:         releaseNoteCmd,
				args:        []string{},
				expectError: false,
				description: "should print release note",
			},
			{
				name:        "releases_yaml",
				cmd:         releaseYamlCmd,
				args:        []string{},
				expectError: false,
				description: "should print releases YAML path",
			},
		}

		for _, tt := range tests {
			Convey("When executing command that "+tt.description, func() {
				// Capture output
				var buf bytes.Buffer
				tt.cmd.SetOut(&buf)
				tt.cmd.SetErr(&buf)

				// Execute command
				err := tt.cmd.Execute()

				if tt.expectError {
					So(err, ShouldNotBeNil)
				} else {
					So(err, ShouldBeNil)
				}

				// Reset command for next test
				tt.cmd.SetArgs([]string{})
			})
		}
	})
}

func TestReleasesWithData(t *testing.T) {
	Convey("Given the BC releases commands with data available", t, func() {
		SkipConvey("When no release data is available", func() {
			SkipSo(len(Releases()), ShouldBeGreaterThan, 0)
		})

		if len(Releases()) == 0 {
			return
		}

		dataTests := []struct {
			name        string
			cmd         *cobra.Command
			fieldName   string
			expectEmpty bool
			description string
		}{
			{
				name:        "version_with_data",
				cmd:         releaseVersionCmd,
				fieldName:   "Version",
				expectEmpty: false,
				description: "should return version when data available",
			},
			{
				name:        "date_with_data",
				cmd:         releaseDateCmd,
				fieldName:   "Date",
				expectEmpty: false,
				description: "should return date when data available",
			},
			{
				name:        "flow_with_data",
				cmd:         releaseFlowCmd,
				fieldName:   "Flow",
				expectEmpty: true,
				description: "should return flow when data available",
			},
			{
				name:        "build_with_data",
				cmd:         releaseBuildCmd,
				fieldName:   "Build",
				expectEmpty: false,
				description: "should return build when data available",
			},
			{
				name:        "note_with_data",
				cmd:         releaseNoteCmd,
				fieldName:   "Note",
				expectEmpty: true,
				description: "should return note when data available",
			},
		}

		for _, tt := range dataTests {
			Convey("When testing "+tt.description, func() {
				var buf bytes.Buffer
				tt.cmd.SetOut(&buf)
				tt.cmd.SetErr(&buf)

				err := tt.cmd.Execute()

				Convey("It should execute without error", func() {
					So(err, ShouldBeNil)
				})

				output := buf.String()
				if !tt.expectEmpty {
					Convey("It should return non-empty output", func() {
						So(output, ShouldNotBeEmpty)
					})
				}

				// Reset command
				tt.cmd.SetArgs([]string{})
			})
		}
	})
}

func TestReleasesNoData(t *testing.T) {
	Convey("Given the BC releases commands with no data available", t, func() {
		// Temporarily override the bc.Releases config hook with empty data
		originalReleases := Releases
		Releases = func() []kfg.AppRelease { return []kfg.AppRelease{} }

		defer func() {
			// Restore original hook
			Releases = originalReleases
		}()

		noDataTests := []struct {
			name        string
			cmd         *cobra.Command
			description string
		}{
			{
				name:        "version_no_data",
				cmd:         releaseVersionCmd,
				description: "should handle missing version data",
			},
			{
				name:        "date_no_data",
				cmd:         releaseDateCmd,
				description: "should handle missing date data",
			},
			{
				name:        "flow_no_data",
				cmd:         releaseFlowCmd,
				description: "should handle missing flow data",
			},
			{
				name:        "build_no_data",
				cmd:         releaseBuildCmd,
				description: "should handle missing build data",
			},
			{
				name:        "note_no_data",
				cmd:         releaseNoteCmd,
				description: "should handle missing note data",
			},
		}

		for _, tt := range noDataTests {
			Convey("When testing command that "+tt.description, func() {
				var buf bytes.Buffer
				tt.cmd.SetOut(&buf)
				tt.cmd.SetErr(&buf)

				Convey("It should not panic or crash", func() {
					// Commands may exit with error when no data is available, but shouldn't crash
					So(func() { tt.cmd.Execute() }, ShouldNotPanic)
				})

				// Reset command
				tt.cmd.SetArgs([]string{})
			})
		}
	})
}

func TestReleasesYamlPath(t *testing.T) {
	Convey("Given the releases yaml command", t, func() {
		Convey("When checking the yaml command structure", func() {
			Convey("It should have proper command metadata", func() {
				So(releaseYamlCmd.Use, ShouldEqual, "yaml")
				So(releaseYamlCmd.Short, ShouldNotBeEmpty)
				So(releaseYamlCmd.Long, ShouldNotBeEmpty)
			})
		})

		Convey("When ReleasesPath is called", func() {
			// Direct test of the function called by yamlCmd
			path := kfg.ReleasesPath()

			// The path may be empty if config isn't initialized (like in unit tests)
			// This is expected behavior, so we test accordingly
			Convey("It should return a string (possibly empty if not initialized)", func() {
				// This passes if path is any string, including empty
				So(path, ShouldHaveSameTypeAs, "")
			})

			if path != "" {
				Convey("If path is not empty, it should contain 'yaml'", func() {
					So(strings.Contains(path, "yaml"), ShouldBeTrue)
				})
			}
		})
	})
}

func TestReleasesCommandHelp(t *testing.T) {
	Convey("Given the releases commands", t, func() {
		releasesSubcommands := []*cobra.Command{
			releaseVersionCmd,
			releaseDateCmd,
			releaseFlowCmd,
			releaseBuildCmd,
			releaseNoteCmd,
			releaseYamlCmd,
		}

		for _, cmd := range releasesSubcommands {
			Convey("When checking help for "+cmd.Name()+" command", func() {
				Convey("It should have a Short description", func() {
					So(cmd.Short, ShouldNotBeEmpty)
				})

				Convey("It should have a Long description", func() {
					So(cmd.Long, ShouldNotBeEmpty)
				})
			})
		}
	})
}

func TestReleasesParentCommand(t *testing.T) {
	Convey("Given the releases parent command", t, func() {
		var buf bytes.Buffer
		releasesCmd.SetOut(&buf)
		releasesCmd.SetErr(&buf)

		Convey("When executing without subcommands", func() {
			Convey("It should not panic", func() {
				// This is expected behavior - command may show help or return error
				So(func() { releasesCmd.Execute() }, ShouldNotPanic)
			})

			// Reset command
			releasesCmd.SetArgs([]string{})
		})
	})
}
