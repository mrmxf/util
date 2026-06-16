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

func TestBCGitCommand(t *testing.T) {
	Convey("Given the BC git command structure", t, func() {

		Convey("When testing the main git command", func() {
			var buf bytes.Buffer
			gitCmd.SetOut(&buf)
			gitCmd.SetErr(&buf)

			Convey("It should have proper command structure", func() {
				So(gitCmd.Use, ShouldEqual, "git")
				So(gitCmd.Short, ShouldNotBeEmpty)
				So(gitCmd.Long, ShouldNotBeEmpty)
			})

			Convey("It should show help when no subcommand is provided", func() {
				err := gitCmd.Execute()
				So(err, ShouldBeNil)
				gitCmd.SetArgs([]string{})
			})
		})

		Convey("When testing git branch command", func() {
			var buf bytes.Buffer
			branchCmd.SetOut(&buf)
			branchCmd.SetErr(&buf)

			Convey("It should have proper command structure", func() {
				So(branchCmd.Use, ShouldEqual, "branch")
				So(branchCmd.Short, ShouldNotBeEmpty)
				So(branchCmd.Long, ShouldNotBeEmpty)
			})

			Convey("In a git repository, it should return the current branch", func() {
				// This test runs in the package folder which is a git repo
				// Check that we can actually see the .git folder in the parent of the bc folder
				_, err := os.Stat("../.git")
				if os.IsNotExist(err) {
					SkipConvey("Not in a git repository - skipping git repo test")
					return
				}

				// Save current directory and change to parent (project root)
				originalDir, _ := os.Getwd()
				err = os.Chdir("..")
				So(err, ShouldBeNil)

				Reset(func() {
					os.Chdir(originalDir)
					buf.Reset()
				})

				// Capture stdout during command execution
				oldStdout := os.Stdout
				r, w, _ := os.Pipe()
				os.Stdout = w

				// Create a fresh copy of branchCmd for testing
				testBranchCmd := &cobra.Command{
					Use:  "branch",
					RunE: branchCmd.RunE,
				}
				// IMPORTANT: Set empty args to prevent Cobra from parsing os.Args (which contains test flags)
				testBranchCmd.SetArgs([]string{})

				done := make(chan string)
				go func() {
					var buf bytes.Buffer
					io.Copy(&buf, r)
					done <- buf.String()
				}()

				err = testBranchCmd.Execute()
				w.Close()
				os.Stdout = oldStdout

				output := <-done
				So(err, ShouldBeNil)
				So(strings.TrimSpace(output), ShouldNotBeEmpty)
			})

			Convey("Outside a git repository, it should handle gracefully", func() {
				// Save current directory and change to /tmp (not a git repo)
				originalDir, _ := os.Getwd()
				err := os.Chdir("/tmp")
				So(err, ShouldBeNil)

				Reset(func() {
					os.Chdir(originalDir)
					buf.Reset()
				})

				testGitCmd := &cobra.Command{Use: "git", Short: "test git"}
				testGitCmd.AddCommand(branchCmd)
				testGitCmd.SetOut(&buf)
				testGitCmd.SetErr(&buf)
				testGitCmd.SetArgs([]string{"branch"})

				err = testGitCmd.Execute()
				So(err, ShouldNotBeNil)
			})
		})

		Convey("When testing git suffix command", func() {
			var buf bytes.Buffer
			suffixCmd.SetOut(&buf)
			suffixCmd.SetErr(&buf)

			Convey("It should have proper command structure", func() {
				So(suffixCmd.Use, ShouldEqual, "suffix")
				So(suffixCmd.Short, ShouldNotBeEmpty)
				So(suffixCmd.Long, ShouldNotBeEmpty)
			})

			Convey("In a git repository", func() {
				// This test runs in the package folder which is a git repo
				// Check that we can actually see the .git folder
				_, err := os.Stat("../.git")
				if os.IsNotExist(err) {
					SkipConvey("Not in a git repository - skipping git repo test")
					return
				}

				// Save current directory and change to parent (project root)
				originalDir, _ := os.Getwd()
				err = os.Chdir("..")
				So(err, ShouldBeNil)

				Reset(func() {
					os.Chdir(originalDir)
					buf.Reset()
				})

				// First get the current branch
				var branchBuf bytes.Buffer
				testBranchCmd := &cobra.Command{Use: "git", Short: "test git"}
				testBranchCmd.AddCommand(branchCmd)
				testBranchCmd.SetOut(&branchBuf)
				testBranchCmd.SetErr(&branchBuf)
				testBranchCmd.SetArgs([]string{"branch"})

				err = testBranchCmd.Execute()
				So(err, ShouldBeNil)
				currentBranch := strings.TrimSpace(branchBuf.String())

				// Now test suffix logic
				testSuffixCmd := &cobra.Command{Use: "git", Short: "test git"}
				testSuffixCmd.AddCommand(suffixCmd)
				testSuffixCmd.SetOut(&buf)
				testSuffixCmd.SetErr(&buf)
				testSuffixCmd.SetArgs([]string{"suffix"})

				err = testSuffixCmd.Execute()
				So(err, ShouldBeNil)

				output := strings.TrimSpace(buf.String())

				Convey("It should return empty string for main branch or branch name for others", func() {
					if currentBranch == "main" {
						So(output, ShouldBeEmpty)
					} else {
						So(output, ShouldEqual, currentBranch)
					}
				})
			})

			Convey("Outside a git repository, it should handle gracefully", func() {
				// Save current directory and change to /tmp (not a git repo)
				originalDir, _ := os.Getwd()
				err := os.Chdir("/tmp")
				So(err, ShouldBeNil)

				Reset(func() {
					os.Chdir(originalDir)
					buf.Reset()
				})

				testSuffixCmd := &cobra.Command{Use: "git", Short: "test git"}
				testSuffixCmd.AddCommand(suffixCmd)
				testSuffixCmd.SetOut(&buf)
				testSuffixCmd.SetErr(&buf)
				testSuffixCmd.SetArgs([]string{"suffix"})

				err = testSuffixCmd.Execute()
				So(err, ShouldNotBeNil)
			})
		})

		Convey("When testing git tag commands", func() {

			Convey("The tag parent command should have proper structure", func() {
				So(tagCmd.Use, ShouldEqual, "tag")
				So(tagCmd.Short, ShouldNotBeEmpty)
				So(tagCmd.Long, ShouldNotBeEmpty)
			})

			Convey("When testing tag head command", func() {
				var buf bytes.Buffer
				headCmd.SetOut(&buf)
				headCmd.SetErr(&buf)

				Convey("It should have proper command structure", func() {
					So(headCmd.Use, ShouldEqual, "head")
					So(headCmd.Short, ShouldNotBeEmpty)
					So(headCmd.Long, ShouldNotBeEmpty)
				})

				if _, err := os.Stat(".git"); !os.IsNotExist(err) {
					Convey("In a git repository, it should execute without crashing", func() {
						err := headCmd.Execute()
						So(err, ShouldBeNil)
						headCmd.SetArgs([]string{})
					})
				} else {
					Convey("Outside a git repository, it should handle gracefully", func() {
						_ = headCmd.Execute()
						// May return error but shouldn't crash
						headCmd.SetArgs([]string{})
					})
				}
			})

			Convey("When testing tag ref command", func() {
				var buf bytes.Buffer
				refCmd.SetOut(&buf)
				refCmd.SetErr(&buf)

				Convey("It should have proper command structure", func() {
					So(refCmd.Use, ShouldEqual, "ref")
					So(refCmd.Short, ShouldNotBeEmpty)
					So(refCmd.Long, ShouldNotBeEmpty)
				})

				Convey("With release data available", func() {
					if len(Releases()) > 0 {
						Convey("It should return the reference version", func() {
							err := refCmd.Execute()
							So(err, ShouldBeNil)

							output := strings.TrimSpace(buf.String())
							So(output, ShouldNotBeEmpty)
							refCmd.SetArgs([]string{})
						})
					} else {
						Convey("Without release data, it should handle gracefully", func() {
							_ = refCmd.Execute()
							// May exit with status 1, but shouldn't crash
							refCmd.SetArgs([]string{})
						})
					}
				})
			})

			Convey("When testing tag tidy command", func() {
				Convey("It should have proper command structure", func() {
					So(tidyCmd.Use, ShouldEqual, "tidy")
					So(tidyCmd.Short, ShouldNotBeEmpty)
					So(tidyCmd.Long, ShouldNotBeEmpty)
				})

				Convey("It should have force flag available", func() {
					forceFlag := tidyCmd.Flags().Lookup("force")
					So(forceFlag, ShouldNotBeNil)
					So(forceFlag.Shorthand, ShouldEqual, "F")
				})
			})
		})

		Convey("When testing git checkout commands", func() {

			Convey("The checkout parent command should have proper structure", func() {
				So(checkoutCmd.Use, ShouldEqual, "checkout")
				So(checkoutCmd.Short, ShouldNotBeEmpty)
				So(checkoutCmd.Long, ShouldNotBeEmpty)
			})

			Convey("When testing checkout list command", func() {
				var buf bytes.Buffer
				listCmd.SetOut(&buf)
				listCmd.SetErr(&buf)

				Convey("It should have proper command structure", func() {
					So(listCmd.Use, ShouldEqual, "list")
					So(listCmd.Short, ShouldNotBeEmpty)
					So(listCmd.Long, ShouldNotBeEmpty)
				})

				if _, err := os.Stat(".git"); !os.IsNotExist(err) {
					Convey("In a git repository, it should list branches and tags", func() {
						err := listCmd.Execute()
						So(err, ShouldBeNil)
						listCmd.SetArgs([]string{})
					})
				} else {
					Convey("Outside a git repository, it should handle gracefully", func() {
						_ = listCmd.Execute()
						// May return error but shouldn't crash
						listCmd.SetArgs([]string{})
					})
				}
			})

			Convey("When testing checkout production command", func() {
				var buf bytes.Buffer
				productionCmd.SetOut(&buf)
				productionCmd.SetErr(&buf)

				Convey("It should have proper command structure", func() {
					So(productionCmd.Use, ShouldEqual, "production")
					So(productionCmd.Short, ShouldNotBeEmpty)
					So(productionCmd.Long, ShouldNotBeEmpty)
				})

				Convey("It should handle execution gracefully", func() {
					_ = productionCmd.Execute()
					// May return error if no production release exists, which is OK
					productionCmd.SetArgs([]string{})
				})
			})
		})

		Convey("When testing command help documentation", func() {
			gitSubcommands := []*cobra.Command{
				branchCmd,
				suffixCmd,
				headCmd,
				refCmd,
				listCmd,
				productionCmd,
				tidyCmd,
			}

			for _, cmd := range gitSubcommands {
				Convey("Command "+cmd.Name()+" should have complete help", func() {
					So(cmd.Short, ShouldNotBeEmpty)
					So(cmd.Long, ShouldNotBeEmpty)
				})
			}
		})

		Convey("When testing command integration", func() {
			Convey("All git subcommands should be registered with git parent", func() {
				expectedSubcommands := []string{
					"branch", "suffix", "tag", "checkout",
				}

				for _, expectedCmd := range expectedSubcommands {
					found := false
					for _, subCmd := range gitCmd.Commands() {
						if subCmd.Name() == expectedCmd {
							found = true
							break
						}
					}
					So(found, ShouldBeTrue)
				}
			})

			Convey("Tag subcommands should be registered with tag parent", func() {
				expectedTagSubcommands := []string{
					"head", "ref", "tidy",
				}

				for _, expectedCmd := range expectedTagSubcommands {
					found := false
					for _, subCmd := range tagCmd.Commands() {
						if subCmd.Name() == expectedCmd {
							found = true
							break
						}
					}
					So(found, ShouldBeTrue)
				}
			})

			Convey("Checkout subcommands should be registered with checkout parent", func() {
				expectedCheckoutSubcommands := []string{
					"list", "production",
				}

				for _, expectedCmd := range expectedCheckoutSubcommands {
					found := false
					for _, subCmd := range checkoutCmd.Commands() {
						if subCmd.Name() == expectedCmd {
							found = true
							break
						}
					}
					So(found, ShouldBeTrue)
				}
			})
		})
	})
}
