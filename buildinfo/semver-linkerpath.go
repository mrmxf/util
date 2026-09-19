//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

//
// manage semantic versions for release.

package buildinfo

import (
	"reflect"
	"runtime"
	"strings"
)

// linkerMarker exists only so reflection can report this package's import
// path, which is where SemVerJSON lives.
type linkerMarker struct{}

// LinkerPath returns the -X path of SemVerJSON for go build -ldflags:
//
//	go build -ldflags "-X github.com/mrmxf/util/buildinfo.SemVerJSON='...'"
//
// It is the path of the variable this package actually reads, whichever app
// imports it. (Before 2026-09 it returned <main module>/semver.SemVerJSON, a
// package clog no longer links, and Go ignores -X for a missing symbol - so
// every build silently kept the "-dev" placeholder.)
func LinkerPath() string {
	if pkg := reflect.TypeOf(linkerMarker{}).PkgPath(); pkg != "" {
		return pkg + ".SemVerJSON"
	}
	return getPackagePath() + ".SemVerJSON"
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
