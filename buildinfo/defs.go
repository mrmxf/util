//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

package buildinfo

import (
	"time"
)

type LinkerDataJSON struct {
	Build    string `json:"build"`  // "prod" for production otherwise dev
	Tag      string `json:"tag"`    // usually `$(git rev-list -1 HEAD)`
	Hash     string `json:"hash"`   // usually `$(git rev-list -1 HEAD)`
	Date     string `json:"date"`   // usually `$(date +%F)`
	Suffix   string `json:"suffix"` // e.g.`rc` applied to VersionInfo.Short
	AppName  string `json:"name"`   // default = basename of `module`  go.mod
	AppTitle string `json:"title"`  // default = basename of `module`  go.mod
}

type VersionInfo struct {
	AppTitle        string `json:"apptitle"` // Command Line Of Go
	AppName         string `json:"appname"`  // clog
	ReleaseCodeName string `json:"codename"` // from releases.yaml
	CommitId        string `json:"id"`       // from linker
	ARCH            string `json:"arch"`     // from linker
	Date            string `json:"date"`     // from linker
	Long            string // made in cleanLinkerData()
	Note            string // from releases.yaml
	OS              string `json:"os"` // from linker
	Short           string // made in cleanLinkerData()
	SuffixLong      string `json:"semverSuffix"` // from linker
	SuffixShort     string // made in cleanLinkerData()
	Tag             string //from releases.yaml
	Err             error  //if an error occurs during init()
}

var IsProductionBuild = false

// linker will override this variable. We parse it at run time
// See the semver package readme for details.
var SemVerJSON = LinkerDataJSONstr

const LinkerDataJSONstr = `{"build":"dev","hash":"","date":"","suffix":"","app":"","title":""}`

//const LinkerDataJSONstr = `{"tag":"v0.9.9","hash":"587be0f365cbcd772504aefe843ecc0a51efbd46","date":"2025-08-06","suffix":"","name":"clog","title":"Command_Line_Of_Go"}`

var dummyTag = "0.0.0"
var dummyHash = "xxxxxXXXXXxxxxxXXXXXxxxxxXXXXXxxxxxXXXXX"
var dummyTime = time.Now().Format("2006-01-02")
var dummySuffix = "-dev"
