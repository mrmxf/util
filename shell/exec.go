//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

// Package shell provides functionality for executing shell commands and managing process output.
package shell

import (
	"io"
	"log/slog"
	"os"
	"os/exec"
)

// ExecControl is a simple structure to  control the output of the [ExecAsync]
// command. See [Exec] for details of how to wrap the Async command to make it
// synchronous.
//
// [ExecAsync] will send the [ExitCode] on the channel when the command
// terminates. No other comms occur on the channel. Note that the [ExitCode]
// will be -1 if the process is running or was terminated by a signal (forced
// termination)
//
// [ExitCode]: https://pkg.go.dev/os#ProcessState.ExitCode
type ExecControl struct {
	StdOutWriter io.Writer
	StdErrWriter io.Writer
	ProcessState chan int
}

// Exec is an asynchronous wrapper for the exec command.
//
//	// run a shell snippet synchronously to list a folder
//	//   nil env - no extra env variables set
//	//   nil ctl - use Stdout & Stderr
//	exe := shell.Exec("/usr/bin/bash", []string{"-c", "ls -al"}, nil, nil)
//	fmt.Printf("Script has returned with exit code %v", exe.ProcessState.ExitCode())
//
// The returned [exec.Cmd] can be used to wait for the command to finish.
func Exec(command string, args []string, env map[string]string, ctlIo *ExecControl) *exec.Cmd {
	ctl := ctlIo
	if ctl == nil {
		ctl = &ExecControl{
			StdOutWriter: os.Stdout,
			StdErrWriter: os.Stderr,
			ProcessState: make(chan int),
		}
		if ctl.ProcessState == nil {
			ctl.ProcessState = make(chan int)
		}
	}
	exe := ExecAsync(command, args, env, ctl)
	// wait for the end of job communication on the channel
	_, open := <-ctl.ProcessState
	if open {
		close(ctl.ProcessState)
	}
	return exe
}

// Execute a shell command asynchronously, restreaming Stdin & StdOut.
//
// To get results from the shell command, use struct [ExecControl]. If ctl
// is nil then Stdout & Stderr will receive the output and the Status
// will be discarded.
//
//	// run a shell snippet Asynchronously to list a folder
//	//   nil env - no extra env variables set
//	//   nil ctl - use Stdout & Stderr
//	exe := shell.Exec("/usr/bin/bash", []string{"-c", "ls -al"}, nil, nil)
//	fmt.Printf("The script is still running...")
//
// The returned [exec.Cmd] can be used to wait for the command to finish.
func ExecAsync(command string, args []string, env map[string]string, ctl *ExecControl) *exec.Cmd {
	exe := exec.Command(command, args...)
	if ctl == nil {
		ctl = &ExecControl{StdOutWriter: os.Stdout, StdErrWriter: os.Stderr}
	}
	// add in any env variables
	exe.Env = os.Environ()
	// append environment variables from the passed map
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
		slog.Error("cmd.Start() failed to start", "command", command, "args", args)
		return exe
	}

	// var wg sync.WaitGroup
	// wg.Add(1)
	// go func() {
	// 	_, errStdout = rewriteStdout(ctl.StdOutWriter, execStdOut)
	// 	_, errStderr = rewriteStdout(ctl.StdErrWriter, execStdErr)
	// 	wg.Done()
	// }()
	// wg.Wait()

	// execute the job and create the control channel
	go func() {
		_, errStdout = rewriteStdout(ctl.StdOutWriter, execStdOut)
		_, errStderr = rewriteStdout(ctl.StdErrWriter, execStdErr)
		ctl.ProcessState <- exe.ProcessState.ExitCode()
		close(ctl.ProcessState)
	}()

	if errStdout != nil {
		slog.Warn("rewriting StdOut during Exec", "command", command, "args", args)
	}
	if errStderr != nil {
		slog.Warn("rewriting Stderr during Exec", "command", command, "args", args)
	}
	return exe
}
