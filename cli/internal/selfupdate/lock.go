package selfupdate

import (
	"os"
	"path/filepath"
	"time"

	"github.com/agentteamland/atl/cli/internal/throttle"
)

const (
	lockName = "upgrade.lock"
	// lockStale: a lock older than this is assumed abandoned (a crashed upgrade)
	// and stolen — an upgrade takes seconds, so this only ever recovers a crash.
	lockStale = 10 * time.Minute
)

// TryLock takes the global self-update lock so a manual `atl upgrade` and the
// session-start auto-apply (or two auto-applies racing) never download+swap at
// once. It is time-based, not pid-based: a lock older than lockStale is treated
// as abandoned and stolen. Returns a release func and whether the lock was
// acquired. It fails open — if the home dir can't be resolved, it returns
// acquired (never block an upgrade on a missing cache path).
func TryLock() (release func(), ok bool) {
	path, err := throttle.StampPath(lockName)
	if err != nil {
		return func() {}, true
	}
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	if fi, err := os.Stat(path); err == nil && time.Since(fi.ModTime()) > lockStale {
		_ = os.Remove(path) // steal a stale lock
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return func() {}, false // held by a live upgrade
	}
	_ = f.Close()
	return func() { _ = os.Remove(path) }, true
}
