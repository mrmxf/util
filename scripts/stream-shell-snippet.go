//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

// Package cmd implements commands for the cobra CLI library

package scripts

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// Execute a shell snippet and get the result, return code and sys error
func CaptureShellSnippet(snippet string, env map[string]string) (string, int, error) {
	// figure out what shell we will run and log it for debugging
	shell := GetShellPath()
	// inContainer := docker.IsRunningInDockerContainer()

	slog.Debug("Capturing shell snippet: ", "shell", shell, "command", snippet) //, "inContainer", inContainer)

	cmd := exec.Command(shell, "-c", snippet)
	cmd.Env = os.Environ()
	// append environemnt variables from the passed map
	for k, v := range env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	stdoutStderr, err := cmd.CombinedOutput()
	exitStatus := cmd.ProcessState.ExitCode()

	//always return the result as though the shell ran it (including logging)
	result := strings.TrimSpace(string(stdoutStderr))

	//some DEBUG logging that will probably break workflows
	slog.Debug("Result of shell snippet: ", "StdOut+StdErr", result, "$?", exitStatus)

	if err != nil {
		return string(stdoutStderr), exitStatus, err
	}
	return result, exitStatus, nil
}

// Execute a shell snippet and stream the result, stdError & return status
func AwaitShellSnippet(snippet string, env map[string]string, cliArgs []string) (int, error) {
	// figure out what shell we will run and log it for debugging
	shell := GetShellPath()
	// inContainer := docker.IsRunningInDockerContainer()

	slog.Debug("Streaming shell snippet: ", "shell", shell, "command", snippet) //, "inContainer", inContainer)

	//append a dummy executable and the arguments so that $1 in the script works.
	args := append([]string{"-c", snippet, "clog(snippet)"}, cliArgs...)
	exitStatus, err := Exec(shell, args, env)

	//some DEBUG logging that will probably break workflows
	slog.Debug("Status of shell snippet: " + fmt.Sprintf("%v", exitStatus))

	return exitStatus, err
}

func init() {
	// log the order of the init files in case there are problems
	_, file, _, _ := runtime.Caller(0)
	slog.Debug("init " + file)
}
