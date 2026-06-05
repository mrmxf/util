//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

//This simple package manages the version number and name.
//
// semver.Info struct is exported for use in an application
//
// The ParseLinkerJson() function initialises the Info struct

package buildinfo

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"log/slog"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"time"
)

// ParseLinkerJSON is a pure function that parses version information from JSON
// without mutating any global state. Returns parsed VersionInfo and whether it's a production build.
func ParseLinkerJSON(semVerJSON string) (VersionInfo, bool, error) {
	// Trim quotes that bash scripts sometimes leave around the JSON
	ldString := strings.Trim(semVerJSON, "\"'")
	slog.Debug("Linker string is (" + ldString + ")")

	// Parse JSON into local LinkerDataJSON struct
	var data LinkerDataJSON
	if err := json.Unmarshal([]byte(ldString), &data); err != nil {
		return VersionInfo{}, false, fmt.Errorf("failed to parse semver JSON: %w", err)
	}
	slog.Debug("semver received", "SemVerJSON", string(ldString))
	slog.Debug("semver parsed  ", "linkerData", data)

	// Determine if production build
	isProd := data.Build == "prod"

	// Validate and fix hash
	if len(data.Hash) != 40 {
		if len(data.Hash) == 0 {
			slog.Debug("WARNING semver Hash has zero length")
		} else {
			slog.Debug("WARNING semver Hash length is invalid", "length", len(data.Hash), "expected", 40)
		}
		data.Hash = dummyHash
	}

	// Validate and fix date
	if _, err := time.Parse("2006-01-02", data.Date); err != nil {
		data.Date = time.Now().Format("2006-01-02")
	}

	// Set app name if empty
	if len(data.AppName) == 0 {
		if bi, ok := debug.ReadBuildInfo(); ok {
			data.AppName = filepath.Base(bi.Main.Path)
		} else {
			data.AppName = "App"
		}
	}

	// Set app title if empty
	if len(data.AppTitle) == 0 {
		if bi, ok := debug.ReadBuildInfo(); ok {
			data.AppTitle = filepath.Base(bi.Main.Path)
		} else {
			data.AppTitle = "AppTitle"
		}
	}

	// Handle suffix for dev builds
	if !isProd {
		if len(data.Suffix) > 0 {
			data.Suffix = fmt.Sprintf("dev-%s", data.Suffix)
		} else {
			data.Suffix = "dev"
		}
	}

	// Build VersionInfo struct
	info := VersionInfo{
		CommitId: data.Hash,
		AppName:  data.AppName,
		AppTitle: strings.ReplaceAll(data.AppTitle, "_", " "),
		Tag:      data.Tag,
		ARCH:     runtime.GOARCH,
		OS:       runtime.GOOS,
		Date:     data.Date,
	}

	// Calculate suffix fields
	if len(data.Suffix) > 0 {
		info.SuffixShort = "-" + data.Suffix
		info.SuffixLong = "-" + data.Suffix + "." + info.CommitId[:4]
	} else {
		info.SuffixShort = ""
		info.SuffixLong = ""
	}

	// Calculate Short and Long version strings
	info.Short = info.Tag + info.SuffixShort
	info.Long = fmt.Sprintf("%s%s (%s:%s:%s)",
		info.Tag,
		info.SuffixLong,
		info.Date,
		info.OS,
		info.ARCH)

	return info, isProd, nil
}

// cleanLinkerData reads the global linker data and mutates global state
// This is kept for backward compatibility with init() and existing code
func cleanLinkerData() error {
	info, isProd, err := ParseLinkerJSON(SemVerJSON)
	if err != nil {
		return err
	}

	// Update global state (for backward compatibility)
	IsProductionBuild = isProd
	parsedInfo = info

	// Also update the global linkerData pointer for any code that might reference it
	linkerData.Build = map[bool]string{true: "prod", false: "dev"}[isProd]
	linkerData.Tag = info.Tag
	linkerData.Hash = info.CommitId
	linkerData.Date = info.Date
	linkerData.AppName = info.AppName
	linkerData.AppTitle = info.AppTitle
	if len(info.SuffixShort) > 0 {
		linkerData.Suffix = strings.TrimPrefix(info.SuffixShort, "-")
	} else {
		linkerData.Suffix = ""
	}

	return nil
}
