//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

//
// manage semantic versions for release.

package buildinfo

var linkerData = &LinkerDataJSON{
	Tag:      dummyTag,
	Hash:     dummyHash,
	Date:     dummyTime,
	Suffix:   dummySuffix,
	AppName:  "",
	AppTitle: "",
}

var parsedInfo VersionInfo

func Info() VersionInfo {
	return parsedInfo
}

// init reads the linker data and exports the values in the VersionInfo Struct
// Maintained for backward compatibility - new code should use ParseLinkerJSON directly
func init() {
	if err := cleanLinkerData(); err != nil {
		parsedInfo.Err = err
	}
	// Short and Long are now calculated in ParseLinkerJSON/cleanLinkerData
}
