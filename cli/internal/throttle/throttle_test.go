package throttle

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestGate(t *testing.T) {
	stamp := filepath.Join(t.TempDir(), "s")

	if !Gate(stamp, 0) {
		t.Fatal("zero interval should always pass")
	}
	if !Gate(stamp, time.Hour) {
		t.Fatal("missing stamp should pass")
	}
	if err := Touch(stamp); err != nil {
		t.Fatal(err)
	}
	if Gate(stamp, time.Hour) {
		t.Fatal("fresh stamp within interval should block")
	}
	time.Sleep(5 * time.Millisecond)
	if !Gate(stamp, time.Millisecond) {
		t.Fatal("stamp older than interval should pass")
	}
}

func TestTouchCreatesDir(t *testing.T) {
	stamp := filepath.Join(t.TempDir(), "nested", "dir", "stamp")
	if err := Touch(stamp); err != nil {
		t.Fatalf("touch: %v", err)
	}
	if _, err := os.Stat(stamp); err != nil {
		t.Fatalf("stamp not created: %v", err)
	}
}
