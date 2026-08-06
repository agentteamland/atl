package retrieve

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func lockFor(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "index.gob")
}

// The reason this exists: two full rebuilds ran concurrently in the wild, each
// taking every core. The second must exit quietly — a duplicate is not an error.
func TestSecondBuildIsRefusedWhileTheFirstHoldsTheLock(t *testing.T) {
	idx := lockFor(t)
	release, ok := AcquireBuildLock(idx)
	if !ok {
		t.Fatal("the first build must get the lock")
	}
	defer release()

	if _, ok := AcquireBuildLock(idx); ok {
		t.Error("a second build must be refused while the first holds the lock")
	}
}

// …and the lock must not outlive the build, or one rebuild would block every
// later one until the file aged out.
func TestReleasingTheLockLetsTheNextBuildIn(t *testing.T) {
	idx := lockFor(t)
	release, ok := AcquireBuildLock(idx)
	if !ok {
		t.Fatal("first acquire")
	}
	release()
	if _, ok := AcquireBuildLock(idx); !ok {
		t.Error("a released lock must not block the next build")
	}
}

// stamped writes a lock naming a specific pid and time, for the cases that have
// to construct a state no ordinary acquire produces.
func stamped(t *testing.T, idx string, pid int, at time.Time) string {
	t.Helper()
	lock := filepath.Join(filepath.Dir(idx), "index.lock")
	if err := os.MkdirAll(filepath.Dir(lock), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(lock, []byte(fmt.Sprintf("%d\n%d\n", pid, at.Unix())), 0o644); err != nil {
		t.Fatal(err)
	}
	return lock
}

// A build killed without releasing must not wedge indexing forever, and the pid
// check alone cannot say so: pids are recycled, so a lock can name a pid that
// now belongs to an unrelated live process. The heartbeat is what separates
// them — a real build refreshes its lock, an inheritor of its pid never does.
func TestALockThatStoppedHeartbeatingIsIgnoredEvenIfItsPidIsAlive(t *testing.T) {
	idx := lockFor(t)
	// Our own pid: unambiguously alive, so only the heartbeat test can free this.
	stamped(t, idx, os.Getpid(), time.Now().Add(-2*buildLockStaleAfter))

	if _, ok := AcquireBuildLock(idx); !ok {
		t.Error("a lock nobody has refreshed must be treated as abandoned")
	}
}

// The regression this fix exists for. The previous version tested age FIRST and
// freed any lock past a fixed max age whatever its pid was doing, so a build
// running longer than that guess was declared dead while demonstrably alive —
// and a second build started on top of it, which is the failure the lock exists
// to prevent. Held now for as long as the holder keeps saying so, which is why
// no constant here encodes "how long a build takes".
func TestAHeartbeatingBuildKeepsItsLockHoweverLongItRuns(t *testing.T) {
	idx := lockFor(t)
	lock := stamped(t, idx, os.Getpid(), time.Now().Add(-99*time.Hour))

	if _, held := readBuildLock(lock); held {
		t.Fatal("precondition: a lock left unrefreshed for 99 hours must read as free")
	}
	// The holder is still working and says so. Nothing about the 99 hours it has
	// been building may override that.
	if err := writeBuildLock(lock); err != nil {
		t.Fatal(err)
	}
	if _, held := readBuildLock(lock); !held {
		t.Error("a build that is still heartbeating must keep its lock regardless of elapsed time")
	}
	if _, ok := AcquireBuildLock(idx); ok {
		t.Error("a second build must be refused while the first is still heartbeating")
	}
}

// The heartbeat itself, which is the actual fix — everything else here only
// checks the predicate that reads its output. A refresh that silently stopped
// firing would leave every other test in this file green while a long build lost
// its lock exactly as before.
func TestTheHolderRefreshesItsLockWhileItBuilds(t *testing.T) {
	prev := buildLockHeartbeat
	buildLockHeartbeat = 5 * time.Millisecond
	t.Cleanup(func() { buildLockHeartbeat = prev })

	idx := lockFor(t)
	release, ok := AcquireBuildLock(idx)
	if !ok {
		t.Fatal("first acquire")
	}
	defer release()

	lock := filepath.Join(filepath.Dir(idx), "index.lock")
	// Age the lock behind the holder's back. A live heartbeat must undo this.
	stamped(t, idx, os.Getpid(), time.Now().Add(-2*buildLockStaleAfter))
	if _, held := readBuildLock(lock); held {
		t.Fatal("precondition: the back-dated lock must read as abandoned")
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if _, held := readBuildLock(lock); held {
			return // the heartbeat refreshed it
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Error("the holder never refreshed its lock — a long build would lose it")
}

// Release must not delete a lock this process does not hold. Without the
// ownership check, the first build finishing removes whichever lock is on disk —
// including one a later build has since taken, handing it to a third.
func TestReleaseDoesNotRemoveAnotherBuildsLock(t *testing.T) {
	idx := lockFor(t)
	release, ok := AcquireBuildLock(idx)
	if !ok {
		t.Fatal("first acquire")
	}
	// Another build takes the lock over while this one is still running.
	other := os.Getpid() + 1
	lock := stamped(t, idx, other, time.Now())

	release()

	pid, _, parsed := parseBuildLock(lock)
	if !parsed {
		t.Fatal("release removed a lock held by another build")
	}
	if pid != other {
		t.Errorf("lock holder = %d, want the other build %d", pid, other)
	}
}

// The other staleness test: a recent lock whose process is gone. Without it a
// build that crashed a minute ago would block indexing for the rest of the hour.
func TestARecentLockWithADeadProcessIsIgnored(t *testing.T) {
	idx := lockFor(t)
	lock := filepath.Join(filepath.Dir(idx), "index.lock")
	if err := os.MkdirAll(filepath.Dir(lock), 0o755); err != nil {
		t.Fatal(err)
	}
	// A pid that cannot be running: the kernel reserves 0, and FindProcess+signal
	// on it never reports a live user process.
	if err := os.WriteFile(lock, []byte(fmt.Sprintf("0\n%d\n", time.Now().Unix())), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, ok := AcquireBuildLock(idx); !ok {
		t.Error("a lock whose process is gone must not block a new build")
	}
}

// A truncated or garbage lock must free the path, not wedge it. The failure mode
// to avoid is a corrupt byte permanently disabling index refresh — silently,
// since nothing reports that retrieval has stopped updating.
func TestAMalformedLockIsTreatedAsFree(t *testing.T) {
	for _, body := range []string{"", "not-a-pid", "12345", "abc\ndef\n", "-1\n0\n"} {
		idx := lockFor(t)
		lock := filepath.Join(filepath.Dir(idx), "index.lock")
		if err := os.MkdirAll(filepath.Dir(lock), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(lock, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, ok := AcquireBuildLock(idx); !ok {
			t.Errorf("malformed lock %q must be treated as free", body)
		}
	}
}

// Best-effort by contract: if the lock cannot be written at all, the build must
// still proceed. An index that silently stops updating is worse than one built
// twice, and nothing surfaces "retrieval went stale".
func TestAnUnwritableLockPathStillAllowsTheBuild(t *testing.T) {
	dir := t.TempDir()
	// A regular file where the lock's directory must be: MkdirAll and WriteFile
	// both fail.
	blocked := filepath.Join(dir, "cache")
	if err := os.WriteFile(blocked, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, ok := AcquireBuildLock(filepath.Join(blocked, "index.gob")); !ok {
		t.Error("an unwritable lock must not prevent indexing")
	}
}

// The lock file may be unwritable — a read-only cache dir, a leftover directory
// at the path. Every such failure proceeds with the build: an index that
// silently stops updating is worse than one built twice, and nothing reports
// that retrieval has gone stale.
func TestAcquireProceedsWhenTheLockCannotBeWritten(t *testing.T) {
	dir := t.TempDir()
	idx := filepath.Join(dir, "index.gob")
	// A directory where the lock file goes: WriteFile fails, os.Stat succeeds.
	if err := os.MkdirAll(filepath.Join(dir, "index.lock"), 0o755); err != nil {
		t.Fatal(err)
	}
	release, ok := AcquireBuildLock(idx)
	if !ok {
		t.Fatal("an unwritable lock must not block the build")
	}
	release() // must not panic, and must not remove the directory's contents
}

// EPERM from signal 0 means the process exists and belongs to someone else —
// alive, not gone. Reading it as "gone" would free a lock held by a live build
// running under another user.
func TestProcessAliveReadsPermissionDeniedAsAlive(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("running as root: no process is permission-denied to us")
	}
	if !processAlive(1) { // pid 1 is root-owned on every unix
		t.Error("a permission-denied process is alive, not gone")
	}
}
