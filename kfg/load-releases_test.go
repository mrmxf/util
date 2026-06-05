//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

package kfg

import (
	"os"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestLoadReleases(t *testing.T) {
	Convey("Given the kfg package", t, func() {
		Convey("When LoadReleases is called with proper test configuration", func() {
			// Create test releases data
			var testReleases []AppRelease

			// Create test releases.yaml file
			testReleasesContent := `- version: "1.0.0"
  date: "2023-01-01"
  flow: "main"
  build: "prod"
  note: "Initial release"
- version: "1.1.0"
  date: "2023-02-01"
  flow: "main"
  build: "prod"
  note: "Feature update"
- version: "0.9.3"
  date: "2022-12-01"
  flow: "main"
  build: "prod"
  note: "core logging fix"`

			// Write test releases file
			err := os.WriteFile("test-releases.yaml", []byte(testReleasesContent), 0644)
			So(err, ShouldBeNil)
			defer os.Remove("test-releases.yaml")

			// Initialize configuration with test releases path
			err = Konfigure(&KonfigureOpt{
				AppFs:               testEfs,
				FilePath:            "konfigure-autoapp_test.yaml",
				PreventAutoMerge:    true,
				PreventAutoApp:      true,
				PreventAutoReleases: true, // We'll test manual LoadReleases
			})
			So(err, ShouldBeNil)

			// Load releases manually
			err = LoadReleases(&testReleases)

			Convey("Then it should load releases without error", func() {
				So(err, ShouldBeNil)
			})

			Convey("And testReleases should be populated", func() {
				So(testReleases, ShouldNotBeNil)
				So(len(testReleases), ShouldEqual, 3)
			})

			Convey("And the first release should have all required fields", func() {
				if len(testReleases) > 0 {
					firstRelease := testReleases[0]
					So(firstRelease.Version, ShouldEqual, "1.0.0")
					So(firstRelease.Date, ShouldEqual, "2023-01-01")
					So(firstRelease.Flow, ShouldEqual, "main")
					So(firstRelease.Build, ShouldEqual, "prod")
					So(firstRelease.Note, ShouldEqual, "Initial release")
				}
			})

			Convey("And releases should contain expected test data", func() {
				// Look for the known test release
				found := false
				for _, release := range testReleases {
					if release.Version == "0.9.3" {
						found = true
						So(release.Flow, ShouldEqual, "main")
						So(release.Build, ShouldEqual, "prod")
						So(release.Note, ShouldEqual, "core logging fix")
						break
					}
				}
				So(found, ShouldBeTrue)
			})
		})

		Convey("When LoadReleases is called without Konfigure", func() {
			// Reset to simulate uninitialized state
			originalRaw := Raw
			Raw = nil

			// Restore after test
			defer func() {
				Raw = originalRaw
			}()

			var testReleases []AppRelease
			err := LoadReleases(&testReleases)

			Convey("Then it should return an error", func() {
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "configuration not initialized")
			})
		})

		Convey("When LoadReleases is called with nil destination", func() {
			// Initialize configuration first
			err := Konfigure(&KonfigureOpt{
				AppFs:               testEfs,
				FilePath:            "konfigure-autoapp_test.yaml",
				PreventAutoMerge:    true,
				PreventAutoApp:      true,
				PreventAutoReleases: true,
			})
			So(err, ShouldBeNil)

			err = LoadReleases(nil)

			Convey("Then it should return without error (nothing to do)", func() {
				So(err, ShouldBeNil)
			})
		})

		Convey("When LoadReleases is called with invalid releases-path", func() {
			// Initialize configuration first
			err := Konfigure(&KonfigureOpt{
				AppFs:               testEfs,
				FilePath:            "konfigure-autoapp_test.yaml",
				PreventAutoMerge:    true,
				PreventAutoApp:      true,
				PreventAutoReleases: true,
			})
			So(err, ShouldBeNil)

			// Override the releases path to a short value directly in Raw
			Raw.Set("kfg.releases-path", "short")

			var testReleases []AppRelease
			err = LoadReleases(&testReleases)

			Convey("Then it should return without error (path too short)", func() {
				So(err, ShouldBeNil)
			})
		})

		Convey("When LoadReleases is called with nonexistent file", func() {
			// Initialize configuration first
			err := Konfigure(&KonfigureOpt{
				AppFs:               testEfs,
				FilePath:            "konfigure-autoapp_test.yaml",
				PreventAutoMerge:    true,
				PreventAutoApp:      true,
				PreventAutoReleases: true,
			})
			So(err, ShouldBeNil)

			// Override the releases path to a nonexistent file
			Raw.Set("kfg.releases-path", "nonexistent-releases.yaml")

			var testReleases []AppRelease
			err = LoadReleases(&testReleases)

			Convey("Then it should return an error (file not found)", func() {
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "no such file")
			})
		})
	})
}
