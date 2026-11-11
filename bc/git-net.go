//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/

package bc

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// gitNetTimeout bounds git operations that touch a remote so a hung or
// unreachable origin cannot wedge a build indefinitely (R4).
const gitNetTimeout = 90 * time.Second

// safeRef rejects a git ref/tag that could be parsed as a git option (a leading
// '-') or contains whitespace, hardening the exec.Command git calls against
// option injection (S1). Refs normally originate from releases.yaml, so this is
// defense-in-depth rather than an untrusted-input boundary.
func safeRef(ref string) error {
	switch {
	case ref == "":
		return fmt.Errorf("empty git ref")
	case strings.HasPrefix(ref, "-"):
		return fmt.Errorf("unsafe git ref %q (leading '-' looks like an option)", ref)
	case strings.ContainsAny(ref, " \t\n"):
		return fmt.Errorf("unsafe git ref %q (whitespace)", ref)
	}
	return nil
}

// gitNet runs a network-touching git subcommand with a timeout and returns its
// stdout. On failure the error carries git's stderr: "exit status 1" alone
// cannot be diagnosed from a CI log.
func gitNet(args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), gitNetTimeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, "git", args...).Output()
	if err != nil {
		var stderr []byte
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			stderr = ee.Stderr
		}
		return out, gitNetErr(ctx, args, err, stderr)
	}
	return out, nil
}

// gitNetRun runs a network-touching git subcommand with a timeout, discarding
// stdout on success. On failure the error carries git's combined output.
func gitNetRun(args ...string) error {
	ctx, cancel := context.WithTimeout(context.Background(), gitNetTimeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, "git", args...).CombinedOutput()
	if err != nil {
		return gitNetErr(ctx, args, err, out)
	}
	return nil
}

// gitNetErr wraps a failed git command's error with the command line, a
// timeout hint and whatever git printed, so the cause reaches the log.
func gitNetErr(ctx context.Context, args []string, err error, output []byte) error {
	cmd := "git " + strings.Join(args, " ")
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return fmt.Errorf("%s: timed out after %s: %w", cmd, gitNetTimeout, err)
	}
	if msg := strings.TrimSpace(string(output)); msg != "" {
		return fmt.Errorf("%s: %w: %s", cmd, err, msg)
	}
	return fmt.Errorf("%s: %w (git printed nothing)", cmd, err)
}
