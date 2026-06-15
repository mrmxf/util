//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

package check

import (
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/mrmxf/util/shell"
)

// Dispatcher runs a clog sub-command in-process and returns its captured
// stdout+stderr and exit code. An app registers one via SetDispatcher so that
// `try:` conditions beginning with "clog " avoid forking a shell. When no
// dispatcher is registered such commands fall back to shell execution (which
// runs the clog binary on PATH), so the package works standalone.
type Dispatcher func(args []string) (output string, exitCode int, err error)

var dispatcher Dispatcher

// SetDispatcher registers the in-process clog dispatcher. Pass nil to clear.
func SetDispatcher(d Dispatcher) { dispatcher = d }

// matchState evaluates a state spec against a captured value and a boolean
// "ok" signal. For env conditions ok = (var set && non-empty); for try
// conditions ok = (exit code == 0). value is the env value or trimmed stdout.
func matchState(state, value string, ok bool) bool {
	state = strings.TrimSpace(state)
	switch {
	case state == "" || state == StateTruthy:
		return ok
	case state == StateFalsy:
		return !ok
	case strings.HasPrefix(state, "~"):
		matched, err := regexp.MatchString(state[1:], strings.TrimSpace(value))
		return err == nil && matched
	case strings.HasPrefix(state, ">"):
		n, errN := strconv.Atoi(strings.TrimSpace(state[1:]))
		v, errV := strconv.Atoi(strings.TrimSpace(value))
		return errN == nil && errV == nil && v > n
	case strings.HasPrefix(state, "<"):
		n, errN := strconv.Atoi(strings.TrimSpace(state[1:]))
		v, errV := strconv.Atoi(strings.TrimSpace(value))
		return errN == nil && errV == nil && v < n
	default:
		// exact literal match against the trimmed value
		return strings.TrimSpace(value) == state
	}
}

// evalEnv evaluates one env condition with pure Go — no shell spawn.
// Returns (passed, value) where value is the env var's contents (for accounting).
func evalEnv(c Condition) (bool, string) {
	val, present := os.LookupEnv(c.Expr)
	ok := present && val != ""
	return matchState(c.State, val, ok), val
}

// evalTry evaluates one try condition. Commands starting with "clog " are
// dispatched in-process when a Dispatcher is registered; otherwise the command
// runs in a shell. Returns (passed, stdout, exitCode).
func evalTry(c Condition, before string, env map[string]string) (bool, string, int) {
	cmd := c.Expr
	if before != "" {
		cmd = before + "\n" + cmd
	}

	var (
		out  string
		exit int
	)
	if fields := strings.Fields(c.Expr); dispatcher != nil && len(fields) > 0 && fields[0] == "clog" {
		// in-process clog dispatch (before: is shell-only, so it is ignored here)
		out, exit, _ = dispatcher(fields[1:])
	} else {
		out, exit, _ = shell.CaptureShellSnippet(cmd, env)
	}

	return matchState(c.State, out, exit == 0), strings.TrimSpace(out), exit
}
