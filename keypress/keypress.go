//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

// return an individual keypress as a string
// optionally allow only keypresses from a restricted vocabulary

package keypress

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/mattn/go-tty"
)

const DBG = false

// in VS code it's hard to debug key sequences
// set TTYDGBstr to the command sequence you want to debug e.g. "^V."
// @ToDO - set this to an ENV variable so that it's easier to keep blank
const TTYDGBstr string = "." //"h."

var TTYDBGidx = 0

func GetKey() string {

	tty, err := tty.Open()
	if err != nil {
		if len(TTYDGBstr) > 0 {
			TTYDBGidx = TTYDBGidx + 1
			return TTYDGBstr[TTYDBGidx-1 : TTYDBGidx]
		}
		// there is no ability to read a dynamic key - fallback to console mode
		reader := bufio.NewReader(os.Stdin)
		//read the next key until NL
		text, _ := reader.ReadString('\n')
		return strings.Trim(text, " \t\r\n")[0:1]
	}
	defer tty.Close()

	r, err := tty.ReadRune()
	if err != nil {
		log.Fatal(err)
	}
	s := string(r)
	if DBG {
		fmt.Printf("key(%s)", s)
	}

	return s
}

// restrict returned keys to a limited vocabulary and provide and optional
//
//	default value that is returned if the ENTER key is pressed
func GetKeyFrom(permittedKeys string, defaultKey string) string {
	if len(defaultKey) == 1 {
		if DBG {
			fmt.Printf("Default key is [%s]", defaultKey)
		}
		permittedKeys = permittedKeys + "\n"
	}
	for {
		r := GetKey()
		if DBG {
			fmt.Printf("[%s]", r)
		}
		if strings.Contains(permittedKeys, r) {
			if r == "\n" && len(defaultKey) == 1 {
				if DBG {
					fmt.Printf("default [%s]'n", r)
				}
				return defaultKey
			}
			return string(r)
		}
	}
}
