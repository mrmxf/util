//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/

package ci

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

// exitErr is a command failure with an exit status, like *exec.ExitError.
type exitErr int

func (e exitErr) Error() string { return fmt.Sprintf("exit status %d", int(e)) }
func (e exitErr) ExitCode() int { return int(e) }

func stubLsRemote(t *testing.T, results ...error) *int {
	t.Helper()
	calls := 0
	origExec, origSleep := execCommand, retrySleep
	execCommand = func(dir, name string, args ...string) (string, error) {
		if name != "git" || args[0] != "ls-remote" {
			t.Fatalf("unexpected command %s %v", name, args)
		}
		err := results[len(results)-1]
		if calls < len(results) {
			err = results[calls]
		}
		calls++
		if err != nil {
			return "fatal: unable to access", err
		}
		return "abc123\trefs/tags/v1.2.3", nil
	}
	retrySleep = func(time.Duration) {}
	t.Cleanup(func() { execCommand, retrySleep = origExec, origSleep })
	return &calls
}

func TestRemoteTagCheckRetriesANetworkBlip(t *testing.T) {
	calls := stubLsRemote(t, exitErr(128), nil)
	if err := remoteTagExists("v1.2.3"); err != nil {
		t.Fatalf("want success after one retry, got %v", err)
	}
	if *calls != 2 {
		t.Errorf("calls = %d, want 2", *calls)
	}
}

func TestRemoteTagCheckMissingTagIsFinal(t *testing.T) {
	calls := stubLsRemote(t, exitErr(2))
	err := remoteTagExists("v1.2.3")
	if err == nil || !strings.Contains(err.Error(), "git push origin v1.2.3") {
		t.Fatalf("want the push hint, got %v", err)
	}
	if *calls != 1 {
		t.Errorf("calls = %d, want 1: origin answered, so no retry", *calls)
	}
}

func TestRemoteTagCheckGivesUp(t *testing.T) {
	calls := stubLsRemote(t, errors.New("could not resolve host"))
	err := remoteTagExists("v1.2.3")
	if err == nil || !strings.Contains(err.Error(), "after 4 attempts") {
		t.Fatalf("want a give-up error, got %v", err)
	}
	if *calls != lsRemoteAttempts {
		t.Errorf("calls = %d, want %d", *calls, lsRemoteAttempts)
	}
}
