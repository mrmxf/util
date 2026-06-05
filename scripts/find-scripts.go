//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

// Package scripts adds support for local bash scripts

package scripts

import (
	"log/slog"
	"os"
	"path/filepath"
	"runtime"

	"github.com/mrmxf/util/crayon"
	"github.com/spf13/cobra"
)

var c = crayon.Color()
var scriptsMap = map[string]map[string]string{}

// Add scripts from clogrc folder
func Bootstrap(rootCmd *cobra.Command, folderGlob string) {

	//look for all shell scripts in the clogrc folder
	scripts, err := filepath.Glob(folderGlob)
	//if there is an error, log it and exit
	if err != nil {
		slog.Error("unable to find scripts "+folderGlob, "err", err)
		os.Exit(1)
	}

	//add each script found
	for _, script := range scripts {
		AddScript(rootCmd, script)
	}
}

func init() {
	// log the order of the init files in case there are problems
	_, file, _, _ := runtime.Caller(0)
	slog.Debug("init " + file)
}
