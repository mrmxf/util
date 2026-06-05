//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

package crayon

const escape = "\x1b"

// CrayonColors is a struct of functions for marking up TTY text.
//
// The target application is making CLI more legible and the roles are based
// on the sort of things that clog does. If you've got a different application
// then fork this repo and define your own
type CrayonColors struct {
	Builtin func(a ...interface{}) string //a builtin like Core
	Command func(a ...interface{}) string //CLI command like godoc
	Debug   func(a ...interface{}) string //debug or de-emphasise something
	Dim     func(a ...interface{}) string //dim or de-emphasise something
	Error   func(a ...interface{}) string //error
	File    func(a ...interface{}) string //file or folder names
	Heading func(a ...interface{}) string //headings
	Info    func(a ...interface{}) string //information messages (not body text)
	Success func(a ...interface{}) string //success
	Text    func(a ...interface{}) string //plain text
	Url     func(a ...interface{}) string //URL  / Uri / links
	Warning func(a ...interface{}) string //Warning
	Xit     func(a ...interface{}) string //stop coloring (used for bash export)

	B func(a ...interface{}) string // shorthand for: Builtin
	C func(a ...interface{}) string // shorthand for: Command
	D func(a ...interface{}) string // shorthand for: Dim & Debug
	E func(a ...interface{}) string // shorthand for: Error
	F func(a ...interface{}) string // shorthand for: File
	H func(a ...interface{}) string // shorthand for: Heading
	I func(a ...interface{}) string // shorthand for: Info
	S func(a ...interface{}) string // shorthand for: Success
	T func(a ...interface{}) string // shorthand for: Text
	U func(a ...interface{}) string // shorthand for: Url
	W func(a ...interface{}) string // shorthand for: Warning
	X func(a ...interface{}) string // shorthand for: Xit

	Amd func(a ...interface{}) string // AMD highlighter
	Arm func(a ...interface{}) string // Arm highlighter
	Lnx func(a ...interface{}) string // Linux highlighter
	Mac func(a ...interface{}) string // Mac highlighter
	Wsm func(a ...interface{}) string // Wasm highlighter
	Win func(a ...interface{}) string // Win highlighter

	//The bg variants all have solid backgrounds
	Bbg func(a ...interface{}) string // background emphasis: Builtin
	Cbg func(a ...interface{}) string // background emphasis: Command
	Dbg func(a ...interface{}) string // background emphasis: Dim / Debug
	Ebg func(a ...interface{}) string // background emphasis: Error
	Fbg func(a ...interface{}) string // background emphasis: File
	Hbg func(a ...interface{}) string // background emphasis: Heading
	Ibg func(a ...interface{}) string // background emphasis: Info
	Sbg func(a ...interface{}) string // background emphasis: Success
	Tbg func(a ...interface{}) string // background emphasis: Text
	Ubg func(a ...interface{}) string // background emphasis: Url
	Wbg func(a ...interface{}) string // background emphasis: Warning
	Xbg func(a ...interface{}) string // background emphasis: Xit
}

var crayonSprint CrayonColors
