// Package throttle is a stamp-file rate limiter — the mechanism behind the
// three-speed cadence. A caller (e.g. the per-prompt tick) checks Gate before
// doing work and Touches the stamp after; a zero interval always passes.
//
// Correctness note: throttling never affects correctness — the queue's dedup
// makes any extra run a no-op. The throttle only bounds how often the work
// actually fires, so the per-prompt hook stays cheap.
package throttle

import (
	"os"
	"path/filepath"
	"time"
)

// Gate reports whether enough time has passed since the stamp at path was last
// touched. A non-positive interval, or a missing stamp, always passes.
func Gate(stampPath string, interval time.Duration) bool {
	if interval <= 0 {
		return true
	}
	info, err := os.Stat(stampPath)
	if err != nil {
		return true // no prior stamp → proceed
	}
	return time.Since(info.ModTime()) >= interval
}

// Touch updates the stamp's modtime, creating it (and its directory) if needed.
func Touch(stampPath string) error {
	if err := os.MkdirAll(filepath.Dir(stampPath), 0o755); err != nil {
		return err
	}
	f, err := os.Create(stampPath)
	if err != nil {
		return err
	}
	return f.Close()
}

// StampPath returns ~/.atl/cache/<name>.
func StampPath(name string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".atl", "cache", name), nil
}
