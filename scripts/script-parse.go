//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

// Package cmd implements commands for the cobra CLI library

package scripts

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"runtime"
	"strings"

	"log/slog"
)

// clog regex to detect a clog command statement
//
//	group[1]="clog"
//	group[2]="command-name"
//	group[3]="[opts]"
//	group[4]="< short help after hash>"
const rexClog = `\s*#\s*(clog)\s*>\s*([a-zA-Z_0-9][a-zA-Z_0-9-]*)(\s+\[opts\]){0,1}\s*(#.*){0,1}`

// short Help override - trailing backslash allows continuations
//
//	group[1]="short"
//	group[2]="<short help>"
const rexShort = `\s*#\s*(short)\s*>\s+(.*)`

// long Help  - trailing backslash allows continuations
//
//	group[1]="extra"
//	group[2]="<long help>"
const rexExtra = `\s*#\s*(extra)\s*>\s+(.*)`

// scan a script file and create a map of the data found
// First 3 lines should be of the format:
//
//		    #  clog> command [opts]    # optional short help
//		    # short> a short help explanation\
//	      #        that can be multi-line using backslashes
//		    # extra> long help printed when --help is used
func ParseScriptInfo(filePath string) (*ScriptInfo, error) {
	inf := ScriptInfo{
		FilePath: filePath,
	}

	file, err := os.Open(filePath)
	if err != nil {
		fmt.Print(err)
		return &inf, err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)

	rClog, _ := regexp.Compile(rexClog)
	rShort, _ := regexp.Compile(rexShort)
	rExtra, _ := regexp.Compile(rexExtra)

	done := false
	count := 0

	//iterate over each script line and check for matches
	for scanner.Scan() {
		line := strings.Trim(scanner.Text(), " \t")
		mClog := rClog.FindStringSubmatch(line)
		mShort := rShort.FindStringSubmatch(line)
		mExtra := rExtra.FindStringSubmatch(line)

		switch {
		case len(mClog) > 1:
			// clog line detected in script
			inf.CmdUse = mClog[2]
			inf.NeedsOpts = (len(mClog[3]) > 0)
			if len(mClog[4]) > 1 {
				inf.CmdShort = strings.TrimSpace(mClog[4])[1:]
			}

		case len(mShort) > 1:
			// Short Help override line
			inf.CmdShort = mShort[2]

			//keep appending continuations until no backslash
			for line[len(line)-1] == '\\' {
				// nuke the trailing backslash
				inf.CmdShort = inf.CmdShort[:len(inf.CmdShort)-2]
				scanner.Scan()
				line = strings.TrimSpace(scanner.Text())
				// break loop if the next line is not a comment
				if line[0] != '#' {
					break
				}
				// short help is a single line, regardless of the script
				inf.CmdShort += " " + strings.TrimSpace(line[1:])
			}
			//deal with corner case of bad continuation
			if inf.CmdShort[len(inf.CmdShort)-1] == '\\' {
				inf.CmdShort = inf.CmdShort[:len(inf.CmdShort)-1]
			}

		case len(mExtra) > 1:
			// "Extra" = Long Help line
			inf.CmdLong = mExtra[2]

			//keep appending continuations until no backslash
			for line[len(line)-1] == '\\' {
				// nuke the trailing backslash
				inf.CmdLong = inf.CmdLong[:len(inf.CmdLong)-2]
				scanner.Scan()
				line = strings.TrimSpace(scanner.Text())
				// break loop if the next line is not a comment
				if line[0] != '#' {
					break
				}
				//long help is multiline but ignores leading space
				inf.CmdLong += "\n" + strings.TrimSpace(line[1:])
			}
			//deal with corner case of bad continuation
			if inf.CmdLong[len(inf.CmdLong)-1] == '\\' {
				inf.CmdLong = inf.CmdLong[:len(inf.CmdLong)-2]
			}
		}

		// check if we have all the metadata or we've parsed our max lines
		done = len(mClog) > 2 && len(mShort) > 2 && len(mExtra) > 2
		count++
		if done || (count >= 10) {
			break
		}

	}

	// warn if there is a command with no help
	if len(inf.CmdUse) > 0 && len(inf.CmdShort) == 0 {
		slog.Warn("script " + c.F(filePath) + " has no short help. Please add:")
		slog.Warn("     # short> some short help text for the menu")
	}

	return &inf, nil
}

func init() {
	// log the order of the init files in case there are problems
	_, file, _, _ := runtime.Caller(0)
	slog.Debug("init " + file)
}
