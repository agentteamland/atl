package selfupdate

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/agentteamland/atl/cli/internal/throttle"
)

const (
	// versionCheckStamp is the ~/.atl/cache stamp that throttles the network check.
	versionCheckStamp = "last-version-check"
	// checkInterval bounds how often the auto-apply hits the network.
	checkInterval = 24 * time.Hour
)

// AutoApply is the session-start entry point for binary self-update. Throttled to
// once per checkInterval, it checks for a newer stable release and, if one exists,
// spawns a DETACHED `atl upgrade` so the download+swap runs independently and the
// NEXT session gets the new binary. It returns a short notice to print (empty when
// there's nothing to say). It never blocks on the download and swallows every
// error — session-start is contractually never-fail.
func AutoApply(ctx context.Context, current string) string {
	// The env brake and dev builds short-circuit before any stamp or network work.
	if os.Getenv(EnvDisable) != "" || current == devVersion || current == "" {
		return ""
	}

	// Throttle: at most one network check per checkInterval, across sessions.
	stamp, err := throttle.StampPath(versionCheckStamp)
	if err != nil {
		return ""
	}
	if !throttle.Gate(stamp, checkInterval) {
		return "" // checked recently
	}
	_ = throttle.Touch(stamp)

	st, err := Check(ctx, current)
	if err != nil || !st.Upgrade {
		return ""
	}

	// Windows can't overwrite a running .exe — notify instead of spawning.
	if runtime.GOOS == "windows" {
		return fmt.Sprintf("atl: %s is available — rerun the install script to upgrade (Windows)", st.Latest)
	}

	if err := spawnDetachedUpgrade(); err != nil {
		return ""
	}
	return fmt.Sprintf("atl: %s available — updating in the background (active next session)", st.Latest)
}
