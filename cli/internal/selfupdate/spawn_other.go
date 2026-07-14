//go:build !unix

package selfupdate

import "errors"

// spawnDetachedUpgrade is unavailable off Unix — a running Windows .exe can't be
// replaced in place. AutoApply guards this path with a runtime.GOOS check and
// notifies the user instead; this stub exists only so the package cross-compiles
// for every goreleaser target.
func spawnDetachedUpgrade() error {
	return errors.New("detached self-update is not supported on this platform")
}
