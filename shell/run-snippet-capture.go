//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

// Package cmd implements commands for the cobra CLI library

package shell

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"
)

// Execute a shell snippet and get the result, return code and sys error
func CaptureShellSnippet(snippet string, env map[string]string) (string, int, error) {
	// figure out what shell we will run and log it for debugging
	shell := GetShellPath()

	slog.Debug("Capturing shell snippet: ", "shell", shell, "command", snippet)

	cmd := exec.Command(shell, "-c", snippet)
	cmd.Env = os.Environ()
	// append environment variables from the passed map
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
func StreamShellSnippet(snippet string, env map[string]string) *exec.Cmd {
	// figure out what shell we will run and log it for debugging
	shell := GetShellPath()

	slog.Debug("Streaming shell snippet: ", "shell", shell, "command", snippet)

	args := []string{"-c", snippet}
	ctl := &ExecControl{StdOutWriter: os.Stdout, StdErrWriter: os.Stderr}
	exitStatus := Exec(shell, args, env, ctl)

	//some DEBUG logging that will probably break workflows
	slog.Debug("Status of shell snippet: " + fmt.Sprintf("%v", exitStatus))
	return exitStatus
}
