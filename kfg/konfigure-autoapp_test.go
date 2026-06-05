//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

package kfg

import (
	"embed"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

//go:embed konfigure-autoapp_test.yaml
var testEfs embed.FS

// TestAppStruct demonstrates comprehensive YAML features for testing
type TestAppStruct struct {
	// Basic types
	Name    string  `testLabel:"name"`
	Version int     `testLabel:"version"`
	Enabled bool    `testLabel:"enabled"`
	Rating  float64 `testLabel:"rating"`

	// Nested struct
	Database DatabaseConfig `testLabel:"database"`

	// Array of strings
	Tags []string `testLabel:"tags"`

	// Array of structs
	Servers []ServerConfig `testLabel:"servers"`

	// Map of strings
	Environment map[string]string `testLabel:"environment"`

	// Map of arrays
	FeatureFlags map[string][]string `testLabel:"feature-flags"`

	// Nested maps
	Metrics map[string]MetricConfig `testLabel:"metrics"`

	// Array of maps
	Endpoints []map[string]interface{} `testLabel:"endpoints"`
}

type DatabaseConfig struct {
	Host     string `testLabel:"host"`
	Port     int    `testLabel:"port"`
	Name     string `testLabel:"name"`
	SSL      bool   `testLabel:"ssl"`
	PoolSize int    `testLabel:"pool-size"`
}

type ServerConfig struct {
	Name     string   `testLabel:"name"`
	URL      string   `testLabel:"url"`
	Regions  []string `testLabel:"regions"`
	Capacity int      `testLabel:"capacity"`
}

type MetricConfig struct {
	Enabled   bool   `testLabel:"enabled"`
	Interval  string `testLabel:"interval"`
	Threshold int    `testLabel:"threshold"`
}

// testApp is the global test configuration instance
var testApp *TestAppStruct

func TestAutoAppUnmarshal(t *testing.T) {
	Convey("Given the kfg package with test configuration", t, func() {
		Convey("When Konfigure is called with custom test configuration", func() {
			// Reset testApp to ensure clean test state
			testApp = nil

			// Configure with test-specific options
			err := Konfigure(&KonfigureOpt{
				AppFs:                  testEfs, // Use test embedded filesystem
				FilePath:               "konfigure-autoapp_test.yaml",
				PreventAutoLoad:        false,
				PreventAutoMerge:       true, // Disable auto-merge for cleaner testing
				PreventAutoApp:         true, // We'll test manual AutoApp
				PreventAutoReleases:    true, // Disable releases for this test
				AutoAppStruct:          &testApp,
				AutoAppKey:             "test-app",
				AutoAppAnnotationLabel: "testLabel",
			})

			Convey("Then configuration should load without error", func() {
				So(err, ShouldBeNil)
			})

			Convey("When AutoAppUnmarshal is called manually", func() {
				err := AutoAppUnmarshal(&KonfigureOpt{
					AutoAppStruct:          &testApp,
					AutoAppKey:             "test-app",
					AutoAppAnnotationLabel: "testLabel",
				})

				Convey("Then it should unmarshal without error", func() {
					So(err, ShouldBeNil)
					So(testApp, ShouldNotBeNil)
				})

				Convey("And basic properties should be populated correctly", func() {
					So(testApp.Name, ShouldEqual, "TestApplication")
					So(testApp.Version, ShouldEqual, 2)
					So(testApp.Enabled, ShouldBeTrue)
					So(testApp.Rating, ShouldEqual, 4.5)
				})

				Convey("And nested struct should be populated", func() {
					So(testApp.Database.Host, ShouldEqual, "localhost")
					So(testApp.Database.Port, ShouldEqual, 5432)
					So(testApp.Database.Name, ShouldEqual, "testdb")
					So(testApp.Database.SSL, ShouldBeTrue)
					So(testApp.Database.PoolSize, ShouldEqual, 10)
				})

				Convey("And array of strings should be populated", func() {
					So(len(testApp.Tags), ShouldEqual, 3)
					So(testApp.Tags[0], ShouldEqual, "development")
					So(testApp.Tags[1], ShouldEqual, "testing")
					So(testApp.Tags[2], ShouldEqual, "golang")
				})

				Convey("And array of structs should be populated", func() {
					So(len(testApp.Servers), ShouldEqual, 2)

					Convey("First server should be correct", func() {
						So(testApp.Servers[0].Name, ShouldEqual, "web-01")
						So(testApp.Servers[0].URL, ShouldEqual, "https://web-01.example.com")
						So(len(testApp.Servers[0].Regions), ShouldEqual, 2)
						So(testApp.Servers[0].Regions[0], ShouldEqual, "us-east-1")
						So(testApp.Servers[0].Regions[1], ShouldEqual, "us-west-2")
						So(testApp.Servers[0].Capacity, ShouldEqual, 100)
					})

					Convey("Second server should be correct", func() {
						So(testApp.Servers[1].Name, ShouldEqual, "api-01")
						So(testApp.Servers[1].URL, ShouldEqual, "https://api-01.example.com")
						So(len(testApp.Servers[1].Regions), ShouldEqual, 1)
						So(testApp.Servers[1].Regions[0], ShouldEqual, "eu-west-1")
						So(testApp.Servers[1].Capacity, ShouldEqual, 200)
					})
				})

				Convey("And map of strings should be populated", func() {
					So(len(testApp.Environment), ShouldEqual, 3)
					So(testApp.Environment["NODE_ENV"], ShouldEqual, "development")
					So(testApp.Environment["LOG_LEVEL"], ShouldEqual, "debug")
					So(testApp.Environment["DEBUG"], ShouldEqual, "true")
				})

				Convey("And map of arrays should be populated", func() {
					So(len(testApp.FeatureFlags), ShouldEqual, 2)

					Convey("auth-methods feature flag should be correct", func() {
						So(len(testApp.FeatureFlags["auth-methods"]), ShouldEqual, 3)
						So(testApp.FeatureFlags["auth-methods"][0], ShouldEqual, "oauth")
						So(testApp.FeatureFlags["auth-methods"][1], ShouldEqual, "saml")
						So(testApp.FeatureFlags["auth-methods"][2], ShouldEqual, "ldap")
					})

					Convey("cache-backends feature flag should be correct", func() {
						So(len(testApp.FeatureFlags["cache-backends"]), ShouldEqual, 2)
						So(testApp.FeatureFlags["cache-backends"][0], ShouldEqual, "redis")
						So(testApp.FeatureFlags["cache-backends"][1], ShouldEqual, "memcached")
					})
				})

				Convey("And nested maps should be populated", func() {
					So(len(testApp.Metrics), ShouldEqual, 2)

					Convey("cpu metric should be correct", func() {
						So(testApp.Metrics["cpu"].Enabled, ShouldBeTrue)
						So(testApp.Metrics["cpu"].Interval, ShouldEqual, "30s")
						So(testApp.Metrics["cpu"].Threshold, ShouldEqual, 80)
					})

					Convey("memory metric should be correct", func() {
						So(testApp.Metrics["memory"].Enabled, ShouldBeTrue)
						So(testApp.Metrics["memory"].Interval, ShouldEqual, "60s")
						So(testApp.Metrics["memory"].Threshold, ShouldEqual, 90)
					})
				})

				Convey("And array of maps should be populated", func() {
					So(len(testApp.Endpoints), ShouldEqual, 2)

					Convey("First endpoint should be correct", func() {
						So(testApp.Endpoints[0]["path"], ShouldEqual, "/api/v1/health")
						So(testApp.Endpoints[0]["method"], ShouldEqual, "GET")
						So(testApp.Endpoints[0]["timeout"], ShouldEqual, 5)
					})

					Convey("Second endpoint should be correct", func() {
						So(testApp.Endpoints[1]["path"], ShouldEqual, "/api/v1/users")
						So(testApp.Endpoints[1]["method"], ShouldEqual, "POST")
						So(testApp.Endpoints[1]["timeout"], ShouldEqual, 30)
					})
				})
			})
		})

		Convey("When AutoAppUnmarshal is called without prior Konfigure", func() {
			// Reset Raw to simulate uninitialized state
			originalRaw := Raw
			Raw = nil

			// Restore Raw after test
			defer func() {
				Raw = originalRaw
			}()

			err := AutoAppUnmarshal(&KonfigureOpt{
				AutoAppStruct:          &testApp,
				AutoAppKey:             "test-app",
				AutoAppAnnotationLabel: "testLabel",
			})

			Convey("Then it should return an error", func() {
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "configuration not initialized")
				So(err.Error(), ShouldContainSubstring, "call Konfigure() before using AutoAppUnmarshal()")
			})
		})

		Convey("When AutoAppUnmarshal is called with nil AutoAppStruct", func() {
			// Initialize configuration first
			err := Konfigure(&KonfigureOpt{
				AppFs:               testEfs,
				FilePath:            "konfigure-autoapp_test.yaml",
				PreventAutoMerge:    true,
				PreventAutoApp:      true,
				PreventAutoReleases: true,
			})
			So(err, ShouldBeNil)

			err = AutoAppUnmarshal(&KonfigureOpt{
				AutoAppStruct:          nil,
				AutoAppKey:             "test-app",
				AutoAppAnnotationLabel: "testLabel",
			})

			Convey("Then it should return without error (nothing to do)", func() {
				So(err, ShouldBeNil)
			})
		})

		Convey("When AutoAppUnmarshal is called with empty AutoAppKey", func() {
			// Initialize configuration first
			err := Konfigure(&KonfigureOpt{
				AppFs:               testEfs,
				FilePath:            "konfigure-autoapp_test.yaml",
				PreventAutoMerge:    true,
				PreventAutoApp:      true,
				PreventAutoReleases: true,
			})
			So(err, ShouldBeNil)

			err = AutoAppUnmarshal(&KonfigureOpt{
				AutoAppStruct:          &testApp,
				AutoAppKey:             "", // Empty key
				AutoAppAnnotationLabel: "testLabel",
			})

			Convey("Then it should return without error (nothing to do)", func() {
				So(err, ShouldBeNil)
			})
		})
	})
}
