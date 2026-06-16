//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/

package bc

import (
	"errors"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Stash writes are a load→modify→write cycle. In CI the stash is touched by many
// separate `clog BC stashLog` *processes*, so an in-process mutex alone cannot
// prevent a lost-update race (R1). withStashLock serializes the critical section
// both in-process (mutex) and cross-process (an advisory <stash>.lock created
// O_EXCL). A lock older than lockStaleAfter is stolen so a crashed holder can't
// wedge the build, and after lockMaxWait we proceed unlocked rather than fail —
// a slightly racy log is better than a failed build.
const (
	lockRetry      = 25 * time.Millisecond
	lockMaxWait    = 10 * time.Second
	lockStaleAfter = 30 * time.Second
)

var stashMu sync.Mutex

func withStashLock(stashPath string, fn func() error) error {
	stashMu.Lock()
	defer stashMu.Unlock()

	lockPath := stashPath + ".lock"
	// The lock lives beside the stash file; make sure that directory exists, or
	// the O_EXCL create below fails with ENOENT (not "lock held") and we'd spin
	// to the timeout on the very first write.
	if dir := filepath.Dir(lockPath); dir != "" {
		_ = os.MkdirAll(dir, 0755)
	}

	deadline := time.Now().Add(lockMaxWait)
	for {
		f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err == nil {
			f.Close()
			defer os.Remove(lockPath)
			return fn()
		}
		if !errors.Is(err, fs.ErrExist) {
			// A genuine problem creating the lock (bad path, perms) — don't spin
			// to the timeout; just proceed unlocked.
			slog.Warn("cannot create stash lock — proceeding unlocked", "lock", lockPath, "error", err)
			return fn()
		}
		// Lock is held: steal it if the holder looks dead (stale), else wait.
		if info, statErr := os.Stat(lockPath); statErr == nil &&
			time.Since(info.ModTime()) > lockStaleAfter {
			slog.Warn("stealing stale stash lock", "lock", lockPath)
			os.Remove(lockPath)
			continue
		}
		if time.Now().After(deadline) {
			slog.Warn("stash lock wait timed out — proceeding unlocked", "lock", lockPath)
			return fn()
		}
		time.Sleep(lockRetry)
	}
}
