//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

//
// manage semantic versions for release.

package buildinfo

import (
	"reflect"
	"runtime"
	"runtime/debug"
	"strings"
)

// LinkerPath returns the linker path string for the SemVerJSON variable
// This path is used with go build -ldflags to set version information at compile time
// Example: go build -ldflags "-X github.com/mrmxf/util/buildinfo.SemVerJSON='...'"
//
// This function dynamically discovers the module path at runtime to construct
// the correct linker path, equivalent to: go tool objdump -S <binary> | grep 'semver.SemVerJSON'
func LinkerPath() string {
	// Try to get the module path dynamically using runtime/debug
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Path != "" {
		return info.Main.Path + "/semver.SemVerJSON"
	}

	// Fallback: Use reflection to get the package path
	// Get the package path of the SemVerJSON variable
	pkgPath := reflect.TypeOf(SemVerJSON).PkgPath()
	if pkgPath == "" {
		// Get package path from this function's location
		pkgPath = getPackagePath()
	}

	// Extract module path from package path
	// For "github.com/mrmxf/util/buildinfo", we want "github.com/mrmxf/clog-mrmxf"
	if idx := strings.LastIndex(pkgPath, "/"); idx != -1 {
		modulePath := pkgPath[:idx]
		return modulePath + "/semver.SemVerJSON"
	}

	// Final fallback to hardcoded path
	return "github.com/mrmxf/util/buildinfo.SemVerJSON"
}

// getPackagePath uses reflection to get the current package path
func getPackagePath() string {
	// Create a dummy function and get its package path
	dummy := func() {}
	pc := reflect.ValueOf(dummy).Pointer()
	fn := runtime.FuncForPC(pc)
	if fn != nil {
		name := fn.Name()
		if idx := strings.LastIndex(name, "."); idx != -1 {
			return name[:idx]
		}
	}
	return "github.com/mrmxf/util/buildinfo"
}
