//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

//
// package cmdlog adds a log command to the clog command line tool

package cmdlog

const helpShort = "Log a message to the configured logger"

const helpLong = helpShort + `
Use the clog/slogger package to create a log message. The log level can be set
in the configuration file or forced to DEBUG with the --debug global option.
All parameters are joined with spaces and the default logger is slogger's
prettyLogger unless overridden in the configuration file`

const helpExample = `
  # messages in ascending logLevel order
	clog Log -T  trace messages require config   "clog.log.level: trace"
	clog Log -D  "debug message"
	clog Log -W  "warning message"
	clog Log -I  info message     joining params   with a single     space
	clog Log -S  "success message"
	clog Log -E  "error message"
	clog Log -F  "fatal message"
	clog Log -X  "emergency message"
	clog Log -I "downloading big file .... (10%)"
	clog Log -UI "downloading big file .... (done) via up/overprint"
`
