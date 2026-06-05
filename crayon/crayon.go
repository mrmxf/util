//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

// Package crayon uses [https://github.com/fatih/color] to provide role base
// colors for logging and highlighting on the console:
//
//   - Builtin
//   - Command
//   - Debug
//   - Dim
//   - Error
//   - File
//   - Heading
//   - Info
//   - Success
//   - Text
//   - Url
//   - Warning
//   - Xit
//
// Roles are defined in [CrayonColors] with a typical usage initialised with
// [Color] and then assigning a shorthand for the few colors you want to use:
//
//	s:= crayon.Color().Success
//	i:= crayon.Color().Info
//	e:= crayon.Color().Error
//	fmt.Printf("%s %s and %s", i("exit with"), s("Success"), e("error"))
//
// Color scheme can be exported to bash/zsh with [GetBashString] and you can
// visualise with [SampleColors].
package crayon

import (
	"fmt"

	"github.com/fatih/color"
)

// ansiExit returns a Sprint() function that prepends the noColor escape seq.
func ansiExit() func(a ...interface{}) string {
	return func(a ...interface{}) string {
		return escape + "[0m" + fmt.Sprint(a...)
	}
}

// Color returns a structure for coloring the ansi output - light Mode.
func Color() *CrayonColors {
	//enable color all the time
	color.NoColor = false

	builtinPlain := color.New(color.FgCyan).Add(color.Bold)
	builtinBlock := color.New(color.BgCyan).Add(color.FgHiYellow).Add(color.Bold)

	commandPlain := color.New(color.FgBlue)
	commandBlock := color.New(color.BgBlue).Add(color.FgYellow)

	dimPlain := color.New(color.FgWhite)
	dimBlock := color.New(color.BgWhite).Add(color.FgBlack)

	errorPlain := color.New(color.FgHiRed)
	errorBlock := color.New(color.BgHiRed).Add(color.FgWhite)

	filePlain := color.New(color.FgYellow)
	fileBlock := color.New(color.BgYellow).Add(color.FgBlack)

	headingPlain := color.New(color.FgHiBlue).Add(color.Bold)
	headingBlock := color.New(color.BgHiBlue).Add(color.FgBlack).Add(color.Bold)

	infoPlain := color.New(color.FgHiYellow)
	infoBlock := color.New(color.BgHiYellow).Add(color.FgBlue)

	successPlain := color.New(color.FgGreen)
	successBlock := color.New(color.BgGreen).Add(color.FgHiYellow)

	textPlain := color.New(color.FgBlack)
	textBlock := color.New(color.BgBlack).Add(color.FgHiWhite)

	urlPlain := color.New(color.FgCyan)
	urlBlock := color.New(color.FgCyan).Add(color.BgCyan)

	warningPlain := color.New(color.FgMagenta)
	warningBlock := color.New(color.FgMagenta).Add(color.BgMagenta)

	crayonSprint.Builtin = builtinPlain.SprintFunc()
	crayonSprint.Command = commandPlain.SprintFunc()
	crayonSprint.Dim = dimPlain.SprintFunc()
	crayonSprint.Error = errorPlain.SprintFunc()
	crayonSprint.File = filePlain.SprintFunc()
	crayonSprint.Heading = headingPlain.SprintFunc()
	crayonSprint.Info = infoPlain.SprintFunc()
	crayonSprint.Success = successPlain.SprintFunc()
	crayonSprint.Text = textPlain.SprintFunc()
	crayonSprint.Url = urlPlain.SprintFunc()
	crayonSprint.Warning = warningPlain.SprintFunc()
	crayonSprint.Xit = ansiExit()

	crayonSprint.B = crayonSprint.Builtin
	crayonSprint.C = crayonSprint.Command
	crayonSprint.D = crayonSprint.Dim
	crayonSprint.E = crayonSprint.Error
	crayonSprint.F = crayonSprint.File
	crayonSprint.H = crayonSprint.Heading
	crayonSprint.I = crayonSprint.Info
	crayonSprint.S = crayonSprint.Success
	crayonSprint.T = crayonSprint.Text
	crayonSprint.U = crayonSprint.Url
	crayonSprint.W = crayonSprint.Warning
	crayonSprint.X = ansiExit()

	crayonSprint.Amd = crayonSprint.Heading
	crayonSprint.Arm = crayonSprint.Success
	crayonSprint.Lnx = crayonSprint.Command
	crayonSprint.Mac = crayonSprint.Warning
	crayonSprint.Win = crayonSprint.Error

	crayonSprint.Bbg = builtinBlock.SprintFunc()
	crayonSprint.Cbg = commandBlock.SprintFunc()
	crayonSprint.Dbg = dimBlock.SprintFunc()
	crayonSprint.Ebg = errorBlock.SprintFunc()
	crayonSprint.Fbg = fileBlock.SprintFunc()
	crayonSprint.Hbg = headingBlock.SprintFunc()
	crayonSprint.Ibg = infoBlock.SprintFunc()
	crayonSprint.Sbg = successBlock.SprintFunc()
	crayonSprint.Tbg = textBlock.SprintFunc()
	crayonSprint.Ubg = urlBlock.SprintFunc()
	crayonSprint.Wbg = warningBlock.SprintFunc()
	crayonSprint.Xbg = ansiExit()

	return &crayonSprint
}
