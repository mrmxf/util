//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

package crayon

import (
	"fmt"
)

// Cursor control helpers using ANSI escape sequences

// Up moves the cursor up by n lines
func Up(n int) string {
	if n <= 0 {
		return ""
	}
	return fmt.Sprintf("\x1b[%dA", n)
}

// Down moves the cursor down by n lines
func Down(n int) string {
	if n <= 0 {
		return ""
	}
	return fmt.Sprintf("\x1b[%dB", n)
}

// Right moves the cursor right by n columns
func Right(n int) string {
	if n <= 0 {
		return ""
	}
	return fmt.Sprintf("\x1b[%dC", n)
}

// Left moves the cursor left by n columns
func Left(n int) string {
	if n <= 0 {
		return ""
	}
	return fmt.Sprintf("\x1b[%dD", n)
}

// Sol moves cursor to start of line (column 0)
func Sol() string {
	return "\x1b[0G"
}

// Eol moves cursor to end of line
// Note: ANSI doesn't have a direct "end of line" command,
// so this moves far right (column 999)
func Eol() string {
	return "\x1b[999C"
}

// ClearLine clears the current line
func ClearLine() string {
	return "\x1b[2K"
}

// ClearToEol clears from cursor to end of line
func ClearToEol() string {
	return "\x1b[0K"
}

// ClearToSol clears from cursor to start of line
func ClearToSol() string {
	return "\x1b[1K"
}

// MoveTo moves cursor to specific row and column (1-indexed)
func MoveTo(row, col int) string {
	return fmt.Sprintf("\x1b[%d;%dH", row, col)
}

// SaveCursor saves the current cursor position
func SaveCursor() string {
	return "\x1b[s"
}

// RestoreCursor restores the cursor to the saved position
func RestoreCursor() string {
	return "\x1b[u"
}

// HideCursor hides the cursor
func HideCursor() string {
	return "\x1b[?25l"
}

// ShowCursor shows the cursor
func ShowCursor() string {
	return "\x1b[?25h"
}

// ClearScreen clears the entire screen
func ClearScreen() string {
	return "\x1b[2J"
}

// Home moves cursor to home position (0,0)
func Home() string {
	return "\x1b[H"
}
