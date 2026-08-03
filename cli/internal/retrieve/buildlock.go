package retrieve

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// buildLockMaxAge bounds how long a lock is believed. A build that dies without
// releasing (SIGKILL, a power cut) must not wedge indexing forever, and the pid
// check alone is not enough: pids are recycled, so a stale lock can name a pid
// that now belongs to something else entirely.
//
// Set well above a realistic full build. The measured worst case on a ~324-page
// corpus is ~17 minutes; an hour leaves room for a much larger corpus on a much
// slower machine without ever making a live build look dead.
const buildLockMaxAge = time.Hour

// AcquireBuildLock takes the index-build lock for a project, returning a release
// function. ok is false when another build already holds it — the caller should
// exit quietly, not error: a second build is a duplicate, not a failure.
//
// This exists because the auto-index path had no single-flight guard at all. It
// throttles how often a build is SPAWNED (10 minutes) but never asked whether
// one was already running, and that gap only became reachable when the corpus
// grew past the point where a build outlasts the throttle window: two full
// rebuilds then run concurrently, each saturating the machine. Measured in the
// wild at 832% CPU with a second process starting behind it.
//
// The lock is advisory and best-effort by design. Every failure path returns
// ok=true — if the lock cannot be written, indexing must still happen; a
// retrieval index that silently stops updating is worse than one built twice.
func AcquireBuildLock(indexPath string) (release func(), ok bool) {
	lockPath := filepath.Join(filepath.Dir(indexPath), "index.lock")
	noop := func() {}

	if holder, alive := readBuildLock(lockPath); alive {
		_ = holder
		return noop, false
	}

	if err := os.MkdirAll(filepath.Dir(lockPath), 0o755); err != nil {
		return noop, true // cannot lock -> proceed rather than skip
	}
	body := fmt.Sprintf("%d\n%d\n", os.Getpid(), time.Now().Unix())
	if err := os.WriteFile(lockPath, []byte(body), 0o644); err != nil {
		return noop, true
	}
	return func() { _ = os.Remove(lockPath) }, true
}

// readBuildLock reports the pid recorded in the lock and whether that build is
// still believed to be running. Two independent staleness tests, because either
// alone is wrong: age catches a lock whose pid was recycled onto an unrelated
// process, and the liveness probe catches a build that died minutes ago and
// would otherwise block indexing for the rest of the hour.
func readBuildLock(lockPath string) (pid int, alive bool) {
	b, err := os.ReadFile(lockPath)
	if err != nil {
		return 0, false // no lock, or unreadable -> treat as free
	}
	fields := strings.Fields(string(b))
	if len(fields) < 2 {
		return 0, false // malformed -> treat as free rather than wedge forever
	}
	pid, err1 := strconv.Atoi(fields[0])
	unix, err2 := strconv.ParseInt(fields[1], 10, 64)
	if err1 != nil || err2 != nil || pid <= 0 {
		return 0, false
	}
	if time.Since(time.Unix(unix, 0)) > buildLockMaxAge {
		return pid, false
	}
	return pid, processAlive(pid)
}

// processAlive reports whether a pid is running. Signal 0 performs the
// permission and existence checks without delivering anything — the portable
// way to ask "is this still there?".
func processAlive(pid int) bool {
	p, err := os.FindProcess(pid) // never fails on unix; the signal is the test
	if err != nil {
		return false
	}
	err = p.Signal(syscall.Signal(0))
	if err == nil {
		return true
	}
	// EPERM means it exists and belongs to someone else — still alive.
	return strings.Contains(err.Error(), "operation not permitted")
}
