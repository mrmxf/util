//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

package kfg

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestUnmarshal(t *testing.T) {
	Convey("Given the kfg package with test configuration", t, func() {
		Convey("When Konfigure is called with test YAML", func() {
			err := Konfigure(&KonfigureOpt{
				AppFs:               testEfs,
				FilePath:            "konfigure-autoapp_test.yaml",
				PreventAutoMerge:    true,
				PreventAutoApp:      true,
				PreventAutoReleases: true,
			})

			Convey("Then configuration should load without error", func() {
				So(err, ShouldBeNil)
			})

			Convey("And when Unmarshal is called for TestAppStruct", func() {
				testConfig, err := Unmarshal[TestAppStruct]("test-app", "testLabel")

				Convey("Then unmarshaling should succeed", func() {
					So(err, ShouldBeNil)
					So(testConfig, ShouldNotBeNil)
				})

				Convey("And basic properties should be populated correctly", func() {
					So(testConfig.Name, ShouldEqual, "TestApplication")
					So(testConfig.Version, ShouldEqual, 2)
					So(testConfig.Enabled, ShouldBeTrue)
					So(testConfig.Rating, ShouldEqual, 4.5)
				})

				Convey("And nested struct should be populated", func() {
					So(testConfig.Database.Host, ShouldEqual, "localhost")
					So(testConfig.Database.Port, ShouldEqual, 5432)
					So(testConfig.Database.Name, ShouldEqual, "testdb")
					So(testConfig.Database.SSL, ShouldBeTrue)
					So(testConfig.Database.PoolSize, ShouldEqual, 10)
				})

				Convey("And array of strings should be populated", func() {
					So(len(testConfig.Tags), ShouldEqual, 3)
					So(testConfig.Tags[0], ShouldEqual, "development")
					So(testConfig.Tags[1], ShouldEqual, "testing")
					So(testConfig.Tags[2], ShouldEqual, "golang")
				})

				Convey("And array of structs should be populated", func() {
					So(len(testConfig.Servers), ShouldEqual, 2)
					So(testConfig.Servers[0].Name, ShouldEqual, "web-01")
					So(testConfig.Servers[0].URL, ShouldEqual, "https://web-01.example.com")
					So(len(testConfig.Servers[0].Regions), ShouldEqual, 2)
					So(testConfig.Servers[0].Regions[0], ShouldEqual, "us-east-1")
					So(testConfig.Servers[0].Regions[1], ShouldEqual, "us-west-2")
					So(testConfig.Servers[0].Capacity, ShouldEqual, 100)
				})

				Convey("And map of strings should be populated", func() {
					So(len(testConfig.Environment), ShouldEqual, 3)
					So(testConfig.Environment["NODE_ENV"], ShouldEqual, "development")
					So(testConfig.Environment["LOG_LEVEL"], ShouldEqual, "debug")
					So(testConfig.Environment["DEBUG"], ShouldEqual, "true")
				})

				Convey("And map of arrays should be populated", func() {
					So(len(testConfig.FeatureFlags), ShouldEqual, 2)
					So(len(testConfig.FeatureFlags["auth-methods"]), ShouldEqual, 3)
					So(testConfig.FeatureFlags["auth-methods"][0], ShouldEqual, "oauth")
					So(testConfig.FeatureFlags["auth-methods"][1], ShouldEqual, "saml")
					So(testConfig.FeatureFlags["auth-methods"][2], ShouldEqual, "ldap")
				})

				Convey("And nested maps should be populated", func() {
					So(len(testConfig.Metrics), ShouldEqual, 2)
					So(testConfig.Metrics["cpu"].Enabled, ShouldBeTrue)
					So(testConfig.Metrics["cpu"].Interval, ShouldEqual, "30s")
					So(testConfig.Metrics["cpu"].Threshold, ShouldEqual, 80)
				})

				Convey("And array of maps should be populated", func() {
					So(len(testConfig.Endpoints), ShouldEqual, 2)
					So(testConfig.Endpoints[0]["path"], ShouldEqual, "/api/v1/health")
					So(testConfig.Endpoints[0]["method"], ShouldEqual, "GET")
					So(testConfig.Endpoints[0]["timeout"], ShouldEqual, 5)
				})
			})
		})
	})
}

func TestUnmarshalWithoutKonfigure(t *testing.T) {
	Convey("Given the kfg package", t, func() {
		Convey("When Unmarshal is called WITHOUT calling Konfigure first", func() {
			// Reset Raw to simulate uninitialized state
			originalRaw := Raw
			Raw = nil

			// Restore Raw after test to avoid affecting other tests
			defer func() {
				Raw = originalRaw
			}()

			// Try to unmarshal without calling Konfigure
			testConfig, err := Unmarshal[TestAppStruct]("test-app", "testLabel")

			Convey("Then it should return an error", func() {
				So(err, ShouldNotBeNil)
				So(testConfig, ShouldBeNil)
			})

			Convey("And the error should indicate Konfigure() needs to be called", func() {
				So(err.Error(), ShouldContainSubstring, "configuration not initialized")
				So(err.Error(), ShouldContainSubstring, "call Konfigure() before using Unmarshal()")
			})
		})
	})
}

func TestMergeKonfig(t *testing.T) {
	Convey("Given the kfg package with configuration loaded", t, func() {
		// First configure with base configuration
		err := Konfigure()
		So(err, ShouldBeNil)

		Convey("When MergeKonfig is called with default options", func() {
			// This will try to load ./.clog.yaml from OS filesystem
			// It's okay if the file doesn't exist - merge is optional
			err := MergeKonfig()

			Convey("Then it should not return an error (even if file doesn't exist)", func() {
				So(err, ShouldBeNil)
			})
		})

		Convey("When MergeKonfig is called without prior Konfigure", func() {
			// Reset Kfg to simulate uninitialized state
			originalKfg := Raw
			Raw = nil

			// Restore Kfg after test
			defer func() {
				Raw = originalKfg
			}()

			err := MergeKonfig()

			Convey("Then it should return an error", func() {
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "configuration not initialized")
				So(err.Error(), ShouldContainSubstring, "call Konfigure() before using MergeKonfig()")
			})
		})
	})
}

func TestAutoMerge(t *testing.T) {
	Convey("Given the kfg package with configuration loaded", t, func() {
		// First configure with base configuration
		err := Konfigure()
		So(err, ShouldBeNil)

		Convey("When AutoMerge is called", func() {
			err := AutoMerge()

			Convey("Then it should process the kfg section", func() {
				// AutoMerge should not error even if files don't exist
				So(err, ShouldBeNil)
			})
		})

		Convey("When AutoMerge is called without prior Konfigure", func() {
			// Reset Kfg to simulate uninitialized state
			originalKfg := Raw
			Raw = nil

			// Restore Kfg after test
			defer func() {
				Raw = originalKfg
			}()

			err := AutoMerge()

			Convey("Then it should return an error", func() {
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "configuration not initialized")
				So(err.Error(), ShouldContainSubstring, "call Konfigure() before using AutoMerge()")
			})
		})
	})
}

func TestKonfigureWithAutoMerge(t *testing.T) {
	Convey("Given the kfg package", t, func() {
		Convey("When Konfigure is called with PreventAutoMerge disabled", func() {
			err := Konfigure(&KonfigureOpt{
				AppFs:               testEfs,
				FilePath:            "konfigure-autoapp_test.yaml",
				PreventAutoMerge:    false,
				PreventAutoApp:      true,
				PreventAutoReleases: true,
			})

			Convey("Then it should load configuration and auto-merge", func() {
				So(err, ShouldBeNil)
				So(Raw, ShouldNotBeNil)

				// Verify we can still unmarshal the test configuration
				testConfig, err := Unmarshal[TestAppStruct]("test-app", "testLabel")
				So(err, ShouldBeNil)
				So(testConfig, ShouldNotBeNil)
				So(testConfig.Name, ShouldEqual, "TestApplication")
			})
		})

		Convey("When Konfigure is called with PreventAutoMerge enabled", func() {
			err := Konfigure(&KonfigureOpt{
				AppFs:               testEfs,
				FilePath:            "konfigure-autoapp_test.yaml",
				PreventAutoMerge:    true,
				PreventAutoApp:      true,
				PreventAutoReleases: true,
			})

			Convey("Then it should load configuration without auto-merge", func() {
				So(err, ShouldBeNil)
				So(Raw, ShouldNotBeNil)
			})
		})
	})
}
