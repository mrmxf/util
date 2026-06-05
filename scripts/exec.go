//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

// Package cmd implements commands for the cobra CLI library

package scripts

import (
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"

	"log/slog"
)

// error codes from https://adminschoice.com/exit-error-codes-in-bash-and-linux-os/
const ERR_ESTRPIPE = 86

// execute a command and restream Stdin & StdOut - return status
func Exec(command string, args []string, env map[string]string) (int, error) {
	exe := exec.Command(command, args...)

	// add in any env variables
	exe.Env = os.Environ()
	// append environemnt variables from the passed map
	for k, v := range env {
		exe.Env = append(exe.Env, k+"="+v)
	}

	// var stdout, stderr []byte
	var errStdout, errStderr error
	execStdOut, _ := exe.StdoutPipe()
	execStdErr, _ := exe.StderrPipe()

	// stdin is unconnected for now - to be debugged
	// exe.Stdin = bufio.NewReader(os.Stdin)
	// stdin, err := cmd.StdinPipe()

	err := exe.Start()
	if err != nil {
		slog.Error("FATAL cmd.Start() during scripts.Exec()")
		slog.Error("FATAL trying to execute", "cmd", strings.Join(args, " "), "err", err)
		return ERR_ESTRPIPE, err
	}

	// cmd.Wait() should be called only after we finish reading
	// from stdoutIn and stderrIn.
	// wg ensures that we finish
	var wg sync.WaitGroup
	var exitCode = 0
	wg.Add(1)
	go func() {
		// defer stdin.Close()
		// io.WriteString(stdin, "values written to stdin are passed to cmd's standard input")

		// stdout, errStdout = copyAndCapture(os.Stdout, stdoutIn)
		_, errStdout = rewriteStdout(os.Stdout, execStdOut)
		_, errStderr = rewriteStdout(os.Stderr, execStdErr)
		wg.Done()
	}()

	// stderr, errStderr = copyAndCapture(os.Stderr, stderrIn)
	_, err = rewriteStdout(os.Stderr, execStdErr)
	if err != nil {
		slog.Error("Failed to rewrite script os.Stderr " + err.Error())
		return ERR_ESTRPIPE, err
	}
	// wait for all the standard output to be rewritten
	wg.Wait()
	//wait for the process to exit
	exe.Wait()
	// grab the exit status of the process
	exitCode = exe.ProcessState.ExitCode()

	if errStdout != nil {
		slog.Warn("WARNING rewriting StdOut")
		slog.Warn("WARNING rewriting StdOut", "cmd", strings.Join(args, " "), "err", errStdout)
	}
	if errStderr != nil {
		slog.Warn("WARNING rewriting Stderr", "cmd", strings.Join(args, " "), "err", errStderr)
	}

	return exitCode, nil
}

func init() {
	// log the order of the init files in case there are problems
	_, file, _, _ := runtime.Caller(0)
	slog.Debug("init " + file)
}
