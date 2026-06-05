//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

//
// pretty printing and other functions for semver

package crayon

import (
	_ "embed"
	"unicode"
)

// iterate through a string and highlight it for display on a TTY.
//
// capital letters at the start of words use the caps highlighter,
// everything else uses the bods highlighter.
func ColorCapitals(str string,
	caps func(a ...interface{}) string,
	bods func(a ...interface{}) string) string {
	var pen = Color()

	if caps == nil {
		caps = pen.Success
	}
	if bods == nil {
		bods = pen.Info
	}

	res := ""
	skipped := ""

	for _, ch := range str {
		if unicode.IsUpper(ch) && unicode.IsLetter(ch) {
			if len(skipped) > 0 {
				res += bods(skipped)
			}
			res += caps(string(ch))
			skipped = ""
		} else {
			skipped = skipped + string(ch)
		}
	}
	if len(skipped) > 0 {
		res += bods(skipped)
	}
	return res
}
