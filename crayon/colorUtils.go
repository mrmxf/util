//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

package crayon

import (
	"fmt"
	"strings"
)

// SampleColors returns a formatted string showing all available colors
func SampleColors() string {
	c := Color()

	msg := ""
	msg = msg + c.H("shell      API         API.long    ") + c.H("Light  ") + "   " + c.H("Inverse") + "\n"
	msg = msg + c.H("-----      ---         --------    ") + c.H("-----  ") + "   " + c.H("-------") + "\n"
	msg = msg + "           c.B        c.Builtin    " + c.B("Builtin") + "   " + c.Bbg("Builtin") + "\n"
	msg = msg + "$cC        c.C        c.Command    " + c.C("Command") + "   " + c.Cbg("Command") + "\n"
	msg = msg + "$cD        c.D        c.Dim        " + c.D("Dim    ") + "   " + c.Dbg("Dim    ") + "\n"
	msg = msg + "$cE        c.E        c.Error      " + c.E("Error  ") + "   " + c.Ebg("Error  ") + "\n"
	msg = msg + "$cF        c.F        c.File       " + c.F("File   ") + "   " + c.Fbg("File   ") + "\n"
	msg = msg + "$cH        c.H        c.Heading    " + c.H("Heading") + "   " + c.Hbg("Heading") + "\n"
	msg = msg + "$cI        c.I        c.Info       " + c.I("Info   ") + "   " + c.Ibg("Info   ") + "\n"
	msg = msg + "$cS        c.S        c.Success    " + c.S("Success") + "   " + c.Sbg("Success") + "\n"
	msg = msg + "$cT        c.T        c.Text       " + c.T("Text   ") + "   " + c.Tbg("Text   ") + "\n"
	msg = msg + "$cU        c.U        c.Url        " + c.U("Url    ") + "   " + c.Ubg("Url    ") + "\n"
	msg = msg + "$cW        c.W        c.Warning    " + c.W("Warning") + "   " + c.Wbg("Warning") + "\n"
	msg = msg + "$cAmd      c.Amd      c.Amd        " + c.Amd("Amd") + "\n"
	msg = msg + "$cArm      c.Arm      c.Arm        " + c.Arm("Arm") + "\n"
	msg = msg + "$cLnx      c.Lnx      c.Lnx        " + c.Lnx("Lnx") + "\n"
	msg = msg + "$cMac      c.Mac      c.Mac        " + c.Mac("Mac") + "\n"
	msg = msg + "$cWin      c.Win      c.Win        " + c.Win("Win") + "\n"
	return msg
}

// toBashStr converts ANSI escape codes to bash-compatible format
func toBashStr(bashVars []string, outputs []string) string {
	// start with the common escape root
	bashStr := ""
	bashEscape := "\\e"
	for i := range bashVars {
		slices := strings.Split(outputs[i], "XXX")
		bashCode := strings.ReplaceAll(slices[0], escape, bashEscape)
		bashStr = fmt.Sprintf("%s%s=\"%s\";", bashStr, bashVars[i], bashCode)
	}
	return bashStr
}

// GetBashString exports color codes as bash-compatible shell variables
func GetBashString(darkMode bool) string {
	c := Color()
	if darkMode {
		c = Color()
	}
	x := "XXX"
	bashVars := []string{"cC", "cE", "cI", "cF", "cH", "cS", "cT", "cU", "cW", "cX", "cAmd", "cArm", "cLnx", "cMac", "cWin"}
	outputs := []string{c.C(x), c.E(x), c.I(x), c.F(x), c.H(x), c.S(x), c.T(x), c.U(x), c.W(x), c.X(x), c.Amd(x), c.Arm(x), c.Lnx(x), c.Mac(x), c.Win(x)}
	return toBashStr(bashVars, outputs)
}
