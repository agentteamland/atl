// Package semver provides a minimal semver comparison used across atl for
// "is version a older than version b" decisions — team-manifest updates and the
// binary self-update "only upgrade, never downgrade" guard. It handles only the
// numeric release triple plus a coarse "a pre-release is older than its final"
// rule (enough for the install→final path) and needs no external dependency. It
// tolerates a leading "v", a "-prerelease" suffix, and "+build" metadata, so a
// stamped buildinfo.Version like "2.3.1" and a release tag like "v2.3.1" compare
// as equal.
package semver

import (
	"strconv"
	"strings"
)

// Less reports whether semver a is strictly older than b. The numeric release
// triple is compared first; on a tie, a pre-release (e.g. 1.0.0-beta) is
// strictly older than the same-numbered final release (1.0.0) — so something
// installed at a pre-release correctly upgrades to its final. Ordering between
// two distinct pre-releases of the same triple is not resolved (they compare
// equal), which is enough for the install→final path.
func Less(a, b string) bool {
	pa, pb := Parse(a), Parse(b)
	for i := 0; i < 3; i++ {
		if pa[i] != pb[i] {
			return pa[i] < pb[i]
		}
	}
	ra, rb := HasPrerelease(a), HasPrerelease(b)
	return ra && !rb // a is a pre-release of the same triple as final b
}

// HasPrerelease reports whether a semver string carries a pre-release segment
// (a "-" after the numeric triple, before any "+build" metadata).
func HasPrerelease(v string) bool {
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	v = strings.SplitN(v, "+", 2)[0]
	return strings.Contains(v, "-")
}

// Parse extracts the numeric release triple from a semver string, tolerating a
// leading "v", a "-prerelease" suffix, and "+build" metadata. Missing or
// non-numeric parts (including the "dev" default of an un-stamped build) become 0.
func Parse(v string) [3]int {
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	v = strings.SplitN(v, "-", 2)[0]
	v = strings.SplitN(v, "+", 2)[0]
	parts := strings.Split(v, ".")
	var out [3]int
	for i := 0; i < 3 && i < len(parts); i++ {
		out[i], _ = strconv.Atoi(parts[i])
	}
	return out
}
