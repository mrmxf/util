//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

// Package scripts provides functionality for parsing and executing script commands.

package scripts

import (
	"fmt"
	"log/slog"
	"os"
	"runtime"

	"github.com/spf13/cobra"
)

const KeywordUse = "clog"
const KeywordShort = "short"
const KeywordLong = "extra"

type ScriptInfo struct {
	CmdUse    string
	CmdShort  string
	CmdLong   string
	NeedsOpts bool
	FilePath  string
}

type ScriptMap map[string]ScriptInfo

var allScripts ScriptMap = make(ScriptMap)

// Add a script from a given filename
func AddScript(cmd *cobra.Command, filePath string) error {

	inf, err := ParseScriptInfo(filePath)
	if err != nil {
		return err
	}
	if len(inf.CmdUse) == 0 {
		// no command found in this script - skip
		return nil
	}
	dupe, ok := allScripts[inf.CmdUse]
	if ok {
		slog.Error(fmt.Sprintf("Script command (%s) duplicated (%s) & (%s)", inf.CmdUse, inf.FilePath, dupe.FilePath))
		return fmt.Errorf("script %s already exists", inf.CmdUse)
	}

	script := &cobra.Command{
		Use: inf.CmdUse,
	}
	if len(inf.CmdShort) > 0 {
		script.Short = inf.CmdShort
	}
	if len(inf.CmdLong) > 0 {
		script.Long = inf.CmdLong
	}
	script.Annotations = map[string]string{
		"command":   cmd.CommandPath() + " " + inf.CmdUse,
		"depth":     "1",
		"is-a":      "script",
		"file-path": inf.FilePath,
		"script":    fmt.Sprintf("eval \"$(cat %s)\"", inf.FilePath),
		"type":      "file",
	}
	script.Run = func(cmd *cobra.Command, args []string) {
		slog.Info(fmt.Sprintf("Script(%s) %s", c.C(inf.CmdUse), c.F(inf.FilePath)))

		shell := []string{inf.FilePath}
		shell = append(shell, args...)
		exitCode, err := Exec("bash", shell, nil)
		if err != nil {
			slog.Debug("Failed to execute script", "err", err.Error())
			os.Exit(1)
		}
		os.Exit(exitCode)
		// exe := exec.Command("bash", shell...)

		// // var stdout, stderr []byte
		// // var errStdout, errStderr error
		// stdoutIn, _ := exe.StdoutPipe()
		// stderrIn, _ := exe.StderrPipe()

		// //connect stdin for console IO
		// exe.Stdin = bufio.NewReader(os.Stdin)
		// err = exe.Start()
		// if err != nil {
		// 	slog.Error("cmd.Start() failed for "+cmd.Name(), "err", err.Error())
		// 	os.Exit(1)
		// }

		// // cmd.Wait() should be called only after we finish reading
		// // from stdoutIn and stderrIn.
		// // wg ensures that we finish
		// var wg sync.WaitGroup
		// wg.Add(1)
		// go func() {
		// 	// stdout, errStdout = copyAndCapture(os.Stdout, stdoutIn)
		// 	_, err = rewriteStdout(os.Stdout, stdoutIn)
		// 	if err != nil {
		// 		slog.Error("Failed to rewrite script os.Stdout " + err.Error())
		// 	}
		// 	wg.Done()
		// }()

		// stderr, errStderr = copyAndCapture(os.Stderr, stderrIn)
		// _, err = rewriteStdout(os.Stderr, stderrIn)
		// if err != nil {
		// 	slog.Error("Failed to rewrite script os.StdErr " + err.Error())
		// }

		// wg.Wait()

		//output has finished, so disconnect StdIn to allow Command to complete
		// exe.Stdin = nil
		// fmt.Println("yo mama")
		// err = exe.Wait()
		// if err != nil {
		// 	slog.Error("cmd.Run() failed for %s with %s\n", cmd.Name(), err)
		// }
		// if (errStdout != nil) || (errStderr != nil) {
		// 	log.Error("failed to capture stdout or stderr\n")
		// }
		// exitStatus := exe.ProcessState.ExitCode()
		// if err != nil || exitStatus > 0 {
		// 	os.Exit(exitStatus)
		// }
	}

	cmd.AddCommand(script)
	allScripts[inf.CmdUse] = *inf
	return nil
}

func init() {
	// log the order of the init files in case there are problems
	_, file, _, _ := runtime.Caller(0)
	slog.Debug("init " + file)
}
