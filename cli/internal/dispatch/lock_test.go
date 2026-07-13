package dispatch

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func TestAcquireRunLockRejectsSecondLive(t *testing.T) {
	root := t.TempDir()
	first, err := acquireRunLock(root)
	if err != nil {
		t.Fatalf("first acquire: %v", err)
	}
	defer first.Release()

	// The current process holds it → a second acquire must fail fast.
	if _, err := acquireRunLock(root); err == nil {
		t.Fatal("a second concurrent acquire must fail while the first is held")
	}

	// After release, the lock is free again.
	first.Release()
	second, err := acquireRunLock(root)
	if err != nil {
		t.Fatalf("acquire after release: %v", err)
	}
	second.Release()
}

func TestAcquireRunLockReclaimsStale(t *testing.T) {
	root := t.TempDir()
	// Simulate a lock left by a crashed run: a pid that is not alive. PID 1 is
	// always alive, so use an implausibly-high pid that won't exist.
	lock := lockPath(root)
	if err := os.MkdirAll(filepath.Dir(lock), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(lock, []byte(strconv.Itoa(0x7FFFFFF0)+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := acquireRunLock(root)
	if err != nil {
		t.Fatalf("a stale lock (dead pid) must be reclaimable, got: %v", err)
	}
	got.Release()
}

func TestAcquireRunLockReclaimsGarbled(t *testing.T) {
	root := t.TempDir()
	lock := lockPath(root)
	if err := os.MkdirAll(filepath.Dir(lock), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(lock, []byte("not-a-pid"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := acquireRunLock(root)
	if err != nil {
		t.Fatalf("a garbled lock must not wedge future runs, got: %v", err)
	}
	got.Release()
}
