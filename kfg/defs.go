//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

// Package kfg provides configuration loading and parsing capabilities for the clog application.
// It uses the koanf library to load YAML configuration from embedded files and unmarshal
// them into Go structs. This package is designed to work with the konfig.yaml configuration
// file that contains settings for the clog CLI tool.
package kfg

import (
	"embed"
	"io/fs"
)

// RootConfig is the filename of the main configuration file within the embedded filesystem.
// This YAML file contains all the default settings for the clog application including
// logging configuration, environment variables, and tool settings.
const RootConfig = "konfig.yaml"

const KongifReleasesPathKey = "kfg.releases-path"

// Efs is an embedded filesystem that contains the configuration files.
// It references the the packages dummy konfig file. Override for your own app
//
//go:embed konfig.yaml
var Efs embed.FS

// KonfigFilePath is the default path for user configuration files that can be merged
// with the main configuration. This defaults to a local .konfig.yaml file.
var KonfigFilePath = ".konfig.yaml"

// KonfigDefaultAppTag is the default label used for unmarshaling the application configuration.
const KonfigDefaultAppTag = "yaml"

// KonfigureOpt provides options for configuring how configuration files are loaded.
// This allows customization of the filesystem source and file paths used for loading.
type KonfigureOpt struct {
	// AppFs specifies the filesystem to load configuration from.
	// For Konfigure: defaults to Efs (embedded filesystem)
	// For MergeKonfig: defaults to OS filesystem
	AppFs fs.FS

	// FilePath is the path to the configuration file within the filesystem.
	// For Konfigure: defaults to RootConfig ("konfig.yaml")
	// For MergeKonfig: defaults to KonfigFilePath (".clog.yaml")
	FilePath string

	// PreventAutoLoad determines whether to automatically load the configuration.
	// Defaults to false. When true, configuration setup is prepared but not AutoLoaded.
	PreventAutoLoad bool

	// PreventAutoMerge determines whether to automatically merge additional configuration files
	// after the main configuration is loaded. When false, automagically reads paths from the "kfg" key
	// and merges each file found in those paths.
	PreventAutoMerge bool

	// PreventAutoApp determines whether to automatically unmarshal the application configuration
	// after the main configuration is loaded. Defaults to false.
	PreventAutoApp bool

	// PreventAutoReleases determines whether to automatically unmarshal the file located at
	// `kfg.releases-path` after the main configuration is loaded. Defaults to false.
	PreventAutoReleases bool

	// AutoAppStruct is a pointer to the destination structure for automatic unmarshaling.
	// If nil, no unmarshalling will take place.
	AutoAppStruct any

	// AutoAppKey is the configuration key to unmarshal from.
	// If not provided, defaults to AppKey ("clog").
	AutoAppKey string

	// AutoAppAnnotationLabel is the label to use for unmarshaling.
	// If not provided, defaults to AppLabel ("koanf").
	AutoAppAnnotationLabel string

	// AutoReleaseSlice is the location for unmarshaling th releases file.
	// If nil, no unmarshalling will take place.
	AutoReleaseSlice *[]AppRelease

	// a pointer to the application args so that a `--debug` flag will provide debug logging
	// make this nil to prevent the functionality
	AppArgs *[]string
}

// defaults according to the comments above
var DefaultKonfigureOpts = KonfigureOpt{
	AppFs:                  Efs,
	FilePath:               RootConfig,
	PreventAutoLoad:        false,
	PreventAutoMerge:       false,
	PreventAutoApp:         false,
	PreventAutoReleases:    false,
	AutoAppStruct:          nil,
	AutoAppKey:             "clog",
	AutoAppAnnotationLabel: "koanf",
	AutoReleaseSlice:       nil,
	AppArgs:                nil,
}

// KonfigPaths represents the structure for configuration paths that should be
// automatically merged. This maps to the "kfg" section in the YAML configuration.
type KonfigPaths struct {
	// TryPaths is an array of file paths that should be attempted for merging.
	// Each path will be tried in order, and missing files are silently ignored.
	TryPaths []string `koanf:"try-paths"`
}

// type AppRelease represents the release information for controlling build
// processes of app, website, packages and various non-golang tools
type AppRelease struct {
	Version string `koanf:"version"`
	Date    string `koanf:"date"`
	Flow    string `koanf:"flow"`
	Build   string `koanf:"build"`
	Note    string `koanf:"note"`
}
