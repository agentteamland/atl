// Package fanout decides how a project-local copy of a shared resource is
// refreshed from the global layer — the pull side of gain circulation.
//
// The rule (decision doc item 5.5): a project copy the user never modified is
// refreshed from global; a copy the user changed locally is preserved — pull,
// never push. "Modified" is a three-way comparison against the hash the file
// was installed as (the manifest baseline), so it means "diverged from what we
// installed", not merely "differs from upstream". This is the same discipline
// v1 used for safe auto-refresh, lifted into a pure, testable core.
package fanout

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
)

// Decision is what to do with one project-local file during a refresh pass.
type Decision int

const (
	// Refresh: local is unmodified (matches the installed baseline) → take the
	// upstream/global version.
	Refresh Decision = iota
	// Preserve: local diverged from the baseline (user-modified) → keep it.
	Preserve
	// UpToDate: local already equals upstream → nothing to do.
	UpToDate
)

func (d Decision) String() string {
	switch d {
	case Refresh:
		return "refresh"
	case Preserve:
		return "preserve"
	default:
		return "up-to-date"
	}
}

// Decide compares the installed baseline hash (what we wrote at install time),
// the current local hash, and the upstream/global hash:
//
//   - local == upstream         → UpToDate (already current; no write)
//   - local == baseline         → Refresh  (unmodified → take upstream)
//   - otherwise (local diverged) → Preserve (user-modified → keep)
//
// UpToDate is checked first so an unmodified file that already matches upstream
// is a no-op rather than a redundant refresh.
func Decide(baseline, local, upstream string) Decision {
	switch {
	case local == upstream:
		return UpToDate
	case local == baseline:
		return Refresh
	default:
		return Preserve
	}
}

// Hash returns the SHA-256 hex of b.
func Hash(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// HashFile returns the SHA-256 hex of the file at path. A missing file hashes
// to "" — so a deleted local copy compares unequal to any baseline and gets
// refreshed.
func HashFile(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return Hash(b), nil
}
