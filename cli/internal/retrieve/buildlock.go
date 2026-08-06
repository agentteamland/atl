package retrieve

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

// The holder refreshes its lock on this interval for as long as it is building,
// and a lock that has not been refreshed for buildLockStaleAfter is considered
// abandoned. Both are properties of the HEARTBEAT, not of a build.
//
// That distinction is the whole point. The previous version bounded the lock by
// a single max age "set well above a realistic full build" — 1 hour, justified
// by a measured worst case of ~17 minutes. That measurement was taken with the
// English embedder; the multilingual model that replaced it is materially slower
// per chunk and the corpus has grown since, so a cold rebuild now runs past the
// hour. The guarantee did not fail because the number was wrong — it failed
// because it was a number at all: any constant of this kind is a duration
// measured in the past with no expiry date, and the subsystem it describes is
// free to move without anyone noticing (this is atl#402's root cause, restated
// one level up).
//
// A heartbeat has no such dependency. It answers "is the holder still working?"
// directly, so a build may take one minute or five hours and the staleness
// window is unchanged.
//
// buildLockHeartbeat is a var rather than a const so a test can shrink it and
// observe the refresh actually happen. Without that seam the goroutine is
// untested by construction — at 30s no unit test would wait for it, and a
// heartbeat that silently stopped firing would look exactly like one that works.
var buildLockHeartbeat = 30 * time.Second

const buildLockStaleAfter = 5 * time.Minute

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

	if _, held := readBuildLock(lockPath); held {
		return noop, false
	}

	if err := os.MkdirAll(filepath.Dir(lockPath), 0o755); err != nil {
		return noop, true // cannot lock -> proceed rather than skip
	}
	if err := writeBuildLock(lockPath); err != nil {
		return noop, true
	}

	// Refresh the lock for as long as this build runs, so its liveness is
	// asserted continuously rather than assumed from a guess about duration.
	// Read the interval HERE, on the caller's goroutine, not inside the one below.
	// Reading a package-level var from the goroutine makes every concurrent
	// caller race with anything that sets it — which the race detector caught on
	// the very test written to exercise this heartbeat.
	every := buildLockHeartbeat
	stop := make(chan struct{})
	go func() {
		tick := time.NewTicker(every)
		defer tick.Stop()
		for {
			select {
			case <-stop:
				return
			case <-tick.C:
				// If the lock is no longer ours, another build has taken it and
				// writing again would silently steal it back.
				if !ownsBuildLock(lockPath) {
					return
				}
				_ = writeBuildLock(lockPath)
			}
		}
	}()

	// release may be called more than once (a defer plus an explicit call);
	// closing an already-closed channel panics, so guard it.
	var once sync.Once
	return func() {
		once.Do(func() {
			close(stop)
			// Remove only our own lock. Without this check, a build finishing
			// deletes whichever lock is on disk — including one a later build
			// has since taken, which hands the lock to a third.
			if ownsBuildLock(lockPath) {
				_ = os.Remove(lockPath)
			}
		})
	}, true
}

// writeBuildLock stamps the lock with this process and the current time. Used
// both to take the lock and to refresh it on the heartbeat.
func writeBuildLock(lockPath string) error {
	body := fmt.Sprintf("%d\n%d\n", os.Getpid(), time.Now().Unix())
	return os.WriteFile(lockPath, []byte(body), 0o644)
}

// ownsBuildLock reports whether the lock on disk still names this process.
func ownsBuildLock(lockPath string) bool {
	pid, _, ok := parseBuildLock(lockPath)
	return ok && pid == os.Getpid()
}

// readBuildLock reports the pid recorded in the lock and whether a build is
// still believed to hold it.
//
// Held means BOTH: the pid is alive, and the lock has been refreshed recently.
// Alive alone is not enough (pids are recycled onto unrelated processes); recent
// alone is not enough (a crashed build leaves a recent lock behind).
//
// What changed here is NOT this predicate — the previous version tested the two
// in the other order and was logically equivalent, so reordering them fixes
// nothing and no comment should claim otherwise. What changed is that the
// timestamp is now REFRESHED by the holder (see AcquireBuildLock). Before, it
// recorded when the build *started*, so a build outliving the max age failed the
// age test while demonstrably alive, and the lock was handed to a second
// concurrent build — the exact failure it exists to prevent. A refreshed stamp
// says "still working" instead of "started long ago", and the two only look the
// same on a build that finishes quickly.
func readBuildLock(lockPath string) (pid int, held bool) {
	pid, stamp, ok := parseBuildLock(lockPath)
	if !ok {
		return 0, false // absent, unreadable or malformed -> treat as free
	}
	if !processAlive(pid) {
		return pid, false
	}
	if time.Since(stamp) > buildLockStaleAfter {
		return pid, false // alive, but not heartbeating -> not the build
	}
	return pid, true
}

// parseBuildLock reads the pid and timestamp a lock records. ok is false for an
// absent, unreadable, truncated or malformed lock — all of which must free the
// path rather than wedge it, since a corrupt byte would otherwise disable index
// refresh permanently and nothing reports that retrieval has stopped updating.
func parseBuildLock(lockPath string) (pid int, stamp time.Time, ok bool) {
	b, err := os.ReadFile(lockPath)
	if err != nil {
		return 0, time.Time{}, false
	}
	fields := strings.Fields(string(b))
	if len(fields) < 2 {
		return 0, time.Time{}, false
	}
	pid, err1 := strconv.Atoi(fields[0])
	unix, err2 := strconv.ParseInt(fields[1], 10, 64)
	if err1 != nil || err2 != nil || pid <= 0 {
		return 0, time.Time{}, false
	}
	return pid, time.Unix(unix, 0), true
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
