//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/

package bc

import (
	"context"
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
// stdout.
func gitNet(args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), gitNetTimeout)
	defer cancel()
	return exec.CommandContext(ctx, "git", args...).Output()
}

// gitNetRun runs a network-touching git subcommand with a timeout, discarding
// stdout.
func gitNetRun(args ...string) error {
	ctx, cancel := context.WithTimeout(context.Background(), gitNetTimeout)
	defer cancel()
	return exec.CommandContext(ctx, "git", args...).Run()
}
