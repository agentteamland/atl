//go:build !unix && !windows

package detach

import "errors"

// Spawn is unavailable on platforms that are neither Unix nor Windows (no
// goreleaser target ships here — this stub exists only so the package
// cross-compiles everywhere). Unix and Windows have real implementations in
// detach_unix.go and detach_windows.go; callers on those platforms use them.
func Spawn(args ...string) error {
	return errors.New("detached spawn is not supported on this platform")
}
