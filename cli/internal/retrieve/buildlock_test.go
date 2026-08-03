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

// A build killed without releasing must not wedge indexing forever. Age is one
// of the two staleness tests, and it is the one that covers a recycled pid — a
// lock naming a pid that now belongs to an unrelated live process.
func TestAnAgedLockIsIgnoredEvenIfItsPidIsAlive(t *testing.T) {
	idx := lockFor(t)
	lock := filepath.Join(filepath.Dir(idx), "index.lock")
	if err := os.MkdirAll(filepath.Dir(lock), 0o755); err != nil {
		t.Fatal(err)
	}
	// Our own pid: unambiguously alive, so only the age test can free this.
	old := time.Now().Add(-2 * buildLockMaxAge).Unix()
	if err := os.WriteFile(lock, []byte(fmt.Sprintf("%d\n%d\n", os.Getpid(), old)), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, ok := AcquireBuildLock(idx); !ok {
		t.Error("a lock older than the max age must be treated as stale")
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
