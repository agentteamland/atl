package dispatch

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

// lockPath is the single-instance lock for a project's dispatch run.
func lockPath(projectRoot string) string {
	return filepath.Join(projectRoot, ".delivery", "dispatch.lock")
}

// runLock is a held single-instance lock; Release removes it.
type runLock struct{ path string }

// acquireRunLock takes an exclusive per-project dispatch lock so a second
// concurrent `atl work dispatch` cannot start: a second supervisor would
// reconcile with an EMPTY active set and quarantine/reclaim the live run's
// worktrees out from under its workers. The lock is a pidfile claimed with
// O_CREATE|O_EXCL (atomic); a lock held by a still-alive pid fails fast, while a
// lock left by a dead pid (a crashed run) is treated as stale and reclaimed.
func acquireRunLock(projectRoot string) (*runLock, error) {
	path := lockPath(projectRoot)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	for attempt := 0; attempt < 2; attempt++ {
		f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
		if err == nil {
			_, _ = f.WriteString(strconv.Itoa(os.Getpid()) + "\n")
			_ = f.Close()
			return &runLock{path: path}, nil
		}
		if !os.IsExist(err) {
			return nil, err
		}
		// Lock present — is its owner still alive?
		pid, live := lockOwnerAlive(path)
		if live {
			return nil, fmt.Errorf("a dispatch is already running for this project (pid %d); lock at %s", pid, path)
		}
		// Stale lock from a crashed run — reclaim it and retry the exclusive create.
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return nil, err
		}
	}
	return nil, fmt.Errorf("could not acquire dispatch lock at %s", path)
}

// lockOwnerAlive reads the pid from the lock file and reports whether that
// process is still running. An unreadable/garbled lock is treated as NOT alive
// (reclaimable) — a corrupt lock must not wedge every future run forever.
func lockOwnerAlive(path string) (int, bool) {
	b, err := os.ReadFile(path)
	if err != nil {
		return 0, false
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(b)))
	if err != nil || pid <= 0 {
		return 0, false
	}
	// Signal 0 probes existence without delivering a signal: nil (or EPERM — the
	// process exists but we can't signal it) means alive; ESRCH means gone.
	err = syscall.Kill(pid, 0)
	if err == nil || err == syscall.EPERM {
		return pid, true
	}
	return pid, false
}

// Release removes the lock file. Safe to call once; a missing file is not an error.
func (l *runLock) Release() {
	if l == nil {
		return
	}
	_ = os.Remove(l.path)
}
