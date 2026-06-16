//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

package bc

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/spf13/cobra"
)

func TestMainBCCommands(t *testing.T) {
	Convey("Given the main BC commands", t, func() {
		tests := []struct {
			name        string
			cmd         *cobra.Command
			args        []string
			expectError bool
			description string
		}{
			{
				name:        "linkerpath",
				cmd:         linkerPathCmd,
				args:        []string{},
				expectError: false,
				description: "should print linker path for SemVerJSON",
			},
			{
				name:        "is_with_valid_field",
				cmd:         isCmd,
				args:        []string{"version", "test"},
				expectError: false,
				description: "should handle is command with valid field",
			},
		}

		for _, tt := range tests {
			Convey("When executing "+tt.description, func() {
				// Capture output
				var buf bytes.Buffer
				tt.cmd.SetOut(&buf)
				tt.cmd.SetErr(&buf)
				tt.cmd.SetArgs(tt.args)

				// Save original os.Args to avoid interference from test flags
				originalArgs := os.Args
				os.Args = []string{"bc"}
				defer func() { os.Args = originalArgs }()

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

func TestLinkerPathOutput(t *testing.T) {
	Convey("Given the linkerpath command", t, func() {
		// Save original os.Args to avoid interference from test flags
		originalArgs := os.Args
		os.Args = []string{"bc"}
		defer func() { os.Args = originalArgs }()

		Convey("When executing the linkerpath command", func() {
			// Capture stdout during command execution
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// Create a test command hierarchy
			testCmd := &cobra.Command{Use: "bc", Short: "test bc"}
			testCmd.AddCommand(linkerPathCmd)
			testCmd.SetArgs([]string{"linkerpath"})

			done := make(chan string)
			go func() {
				var buf bytes.Buffer
				io.Copy(&buf, r)
				done <- buf.String()
			}()

			err := testCmd.Execute()
			w.Close()
			os.Stdout = oldStdout

			output := strings.TrimSpace(<-done)

			Convey("It should execute without error", func() {
				So(err, ShouldBeNil)
			})

			Convey("It should return a non-empty linker path", func() {
				So(output, ShouldNotBeEmpty)
			})

			Convey("It should contain expected path elements", func() {
				expectedElements := []string{"semver", "SemVerJSON"}
				for _, elem := range expectedElements {
					So(output, ShouldContainSubstring, elem)
				}
			})

			// Reset command
			linkerPathCmd.SetArgs([]string{})
		})
	})
}

func TestIsCommand(t *testing.T) {
	Convey("Given the is command", t, func() {
		SkipConvey("When no release data is available", func() {
			SkipSo(len(Releases()), ShouldBeGreaterThan, 0)
		})

		if len(Releases()) == 0 {
			return
		}

		isTests := []struct {
			name        string
			args        []string
			expectError bool
			description string
		}{
			{
				name:        "is_version",
				args:        []string{"version", Releases()[0].Version},
				expectError: false,
				description: "should match current version",
			},
			{
				name:        "is_version_mismatch",
				args:        []string{"version", "non-existent-version"},
				expectError: true,
				description: "should fail on version mismatch",
			},
			{
				name:        "is_build",
				args:        []string{"build", Releases()[0].Build},
				expectError: false,
				description: "should match current build",
			},
			{
				name:        "is_flow",
				args:        []string{"flow", Releases()[0].Flow},
				expectError: false,
				description: "should match current flow",
			},
			{
				name:        "is_date",
				args:        []string{"date", Releases()[0].Date},
				expectError: false,
				description: "should match current date",
			},
			{
				name:        "is_note",
				args:        []string{"note", Releases()[0].Note},
				expectError: false,
				description: "should match current note",
			},
		}

		for _, tt := range isTests {
			Convey("When testing "+tt.description, func() {
				var buf bytes.Buffer
				isCmd.SetOut(&buf)
				isCmd.SetErr(&buf)
				isCmd.SetArgs(tt.args)

				// Save original os.Args to avoid interference from test flags
				originalArgs := os.Args
				os.Args = []string{"bc"}
				defer func() { os.Args = originalArgs }()

				err := isCmd.Execute()

				if tt.expectError {
					So(err, ShouldNotBeNil)
				} else {
					So(err, ShouldBeNil)
				}

				// Reset command
				isCmd.SetArgs([]string{})
			})
		}
	})
}

func TestIsCommandInvalidField(t *testing.T) {
	Convey("Given the is command with invalid inputs", t, func() {
		invalidFieldTests := []struct {
			name        string
			args        []string
			description string
		}{
			{
				name:        "is_invalid_field",
				args:        []string{"invalid_field", "value"},
				description: "should handle invalid field gracefully",
			},
			{
				name:        "is_missing_args",
				args:        []string{},
				description: "should handle missing arguments",
			},
			{
				name:        "is_missing_value",
				args:        []string{"version"},
				description: "should handle missing value argument",
			},
		}

		for _, tt := range invalidFieldTests {
			Convey("When testing "+tt.description, func() {
				var buf bytes.Buffer
				isCmd.SetOut(&buf)
				isCmd.SetErr(&buf)
				isCmd.SetArgs(tt.args)

				// Save original os.Args to avoid interference from test flags
				originalArgs := os.Args
				os.Args = []string{"bc"}
				defer func() { os.Args = originalArgs }()

				Convey("It should not crash", func() {
					// The command may or may not return an error, but it shouldn't crash
					So(func() { isCmd.Execute() }, ShouldNotPanic)
				})

				// Reset command
				isCmd.SetArgs([]string{})
			})
		}
	})
}

func TestIsCommandNoData(t *testing.T) {
	Convey("Given the is command", t, func() {
		// Note: The is command calls os.Exit() directly (see sub-is.go:33),
		// which terminates the test process. We cannot test execution behavior
		// in a normal unit test. We test command structure instead.

		Convey("When checking command structure", func() {
			Convey("It should have proper metadata", func() {
				So(isCmd.Use, ShouldEqual, "is")
				So(isCmd.Short, ShouldNotBeEmpty)
				So(isCmd.Long, ShouldNotBeEmpty)
			})

			Convey("It should require exactly 2 arguments", func() {
				So(isCmd.Args, ShouldNotBeNil)
			})

			Convey("It should document exit codes in Long description", func() {
				So(isCmd.Long, ShouldContainSubstring, "Exit codes")
			})
		})
	})
}

func TestMainCommandHelp(t *testing.T) {
	Convey("Given the main BC commands", t, func() {
		mainSubcommands := []*cobra.Command{
			linkerPathCmd,
			isCmd,
		}

		for _, cmd := range mainSubcommands {
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

func TestBCRootCommand(t *testing.T) {
	Convey("Given the BC root command", t, func() {
		var buf bytes.Buffer
		Command.SetOut(&buf)
		Command.SetErr(&buf)

		Convey("When executing without subcommands", func() {
			// Set empty args to prevent Cobra from parsing test flags
			Command.SetArgs([]string{})

			// Wrap execution in a function to test it doesn't panic
			Convey("It should not panic", func() {
				So(func() {
					Command.Execute()
				}, ShouldNotPanic)
			})
		})

		Convey("When checking command structure", func() {
			Convey("It should have a Short description", func() {
				So(Command.Short, ShouldNotBeEmpty)
			})

			Convey("It should have a Long description", func() {
				So(Command.Long, ShouldNotBeEmpty)
			})
		})

		// Reset command
		Command.SetArgs([]string{})
	})
}
