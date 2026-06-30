package guard

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
)

// FirstEditFunc returns a predicate that reports whether path is being edited for
// the FIRST time in the given session, recording it so later calls return false.
//
// State is per-session marker files under ~/.atl/cache/guard/<session>/, mirroring
// the throttle/cache idiom. On any error (no HOME, an unwritable cache) it returns
// false, suppressing the nudge — guard must never block or spam on its own failure.
func FirstEditFunc(sessionID string) func(path string) bool {
	return func(path string) bool {
		if sessionID == "" || path == "" {
			return false
		}
		dir, err := sessionDir(sessionID)
		if err != nil {
			return false
		}
		marker := filepath.Join(dir, hashPath(path))
		if _, err := os.Stat(marker); err == nil {
			return false // already edited this file this session
		}
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return false
		}
		if err := os.WriteFile(marker, nil, 0o644); err != nil {
			return false
		}
		return true
	}
}

// sessionDir is ~/.atl/cache/guard/<sanitized session id>.
func sessionDir(sessionID string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".atl", "cache", "guard", sanitize(sessionID)), nil
}

// hashPath keys a marker by the file path without embedding the path (which may
// contain separators) in the filename.
func hashPath(p string) string {
	sum := sha256.Sum256([]byte(p))
	return hex.EncodeToString(sum[:])
}

// sanitize keeps a session id safe as a single directory name (session ids are
// UUIDs, but be defensive against path traversal).
func sanitize(s string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			return r
		default:
			return '_'
		}
	}, s)
}
