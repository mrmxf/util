//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

// package semver_test tries to make it hard to accidentally break
// backwards compatibility in the package

package buildinfo_test

import (
	"time"
)

type refLinkerData struct {
	BuildHash           string // usually `$(git rev-list -1 HEAD)`
	BuildDate           string // usually `$(date +%F)`
	BuildSemanticSuffix string // e.g.`rc` applied to VersionInfo.Short
	BuildAppName        string // default = basename of `module`  go.mod
	BuildAppTitle       string // default = basename of `module`  go.mod
}

type refVersionInfo struct {
	AppTitle    string `json:"apptitle"` // Command Line Of Go
	AppName     string `json:"appname"`  // clog
	CodeName    string `json:"codename"` // from releases.yaml
	CommitId    string `json:"id"`       // from linker
	ARCH        string `json:"arch"`     // from linker
	Date        string `json:"date"`     // from linker
	Long        string // made in cleanLinkerData()
	Note        string // from releases.yaml
	OS          string `json:"os"` // from linker
	Short       string // made in cleanLinkerData()
	SuffixLong  string `json:"semverSuffix"` // from linker
	SuffixShort string // made in cleanLinkerData()
	Version     string //from releases.yaml
}

// JSON & YAML field names are the same
type refReleaseHistory struct {
	Appname  string    `json:"appname"`
	Version  string    `json:"version"`
	Date     time.Time `json:"date"`
	CodeName string    `json:"codename"`
	Note     string    `json:"note"`
}
