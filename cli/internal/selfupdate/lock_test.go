package selfupdate

import (
	"os"
	"testing"
	"time"

	"github.com/agentteamland/atl/cli/internal/throttle"
)

func TestTryLock(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	release, ok := TryLock()
	if !ok {
		t.Fatal("first TryLock should acquire")
	}
	if _, ok2 := TryLock(); ok2 {
		t.Error("second TryLock should fail while the lock is held")
	}
	release()
	release2, ok3 := TryLock()
	if !ok3 {
		t.Error("TryLock should acquire again after release")
	}
	release2()
}

func TestTryLockStealsStale(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	// Acquire but do NOT release, then backdate the lock past lockStale.
	if _, ok := TryLock(); !ok {
		t.Fatal("initial acquire failed")
	}
	path, err := throttle.StampPath(lockName)
	if err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-2 * lockStale)
	if err := os.Chtimes(path, old, old); err != nil {
		t.Fatal(err)
	}

	release, ok := TryLock()
	if !ok {
		t.Error("a stale lock should be stolen")
	}
	release()
}
