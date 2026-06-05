//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

// Package cmd implements commands for the cobra CLI library

package shell

import (
	"fmt"
	"os"
)

// Execute a shell snippet, print & return result
// On error (exitStatus>0), and os.Exit(exitStatus)
func ShellSnippet(snippet string) string {
	result, exitStatus, err := CaptureShellSnippet(snippet, nil)

	fmt.Print(result)

	if err != nil || exitStatus > 0 {
		os.Exit(exitStatus)
	}
	return result
}
