package commands

import (
	"os"
	"time"

	"github.com/agentteamland/atl/cli/internal/detach"
	"github.com/agentteamland/atl/cli/internal/throttle"
)

const (
	// envNoTeamUpdate turns the automatic team-update off entirely — the only
	// opt-out, matching selfupdate's ATL_NO_SELF_UPDATE brake for the binary half
	// (the v2 "automation is mandatory" stance).
	envNoTeamUpdate = "ATL_NO_TEAM_UPDATE"
	// teamUpdateInterval bounds how often the auto team-update hits the network,
	// matching the binary self-update's 24h cadence.
	teamUpdateInterval = 24 * time.Hour
)

// teamUpdateSpawn detaches `atl update` so the (possibly slow) index refresh +
// team re-fetch runs independently of the session-start process. A package var
// so tests can observe the throttle/brake decision without forking a real
// process.
var teamUpdateSpawn = func() error { return detach.Spawn("update") }

// autoUpdateTeams is the session-start entry point for the automatic network
// team-update — the "manual `atl update` becomes unnecessary" v2 promise, and
// the team-asset sibling of selfupdate.AutoApply's binary half. Throttled to
// once per teamUpdateInterval per project, it spawns a DETACHED `atl update` so
// the network work never blocks the SessionStart hook; the next session sees any
// newly-published team versions. It runs automatically and silently (no notice)
// and swallows every error — a hook must never block or fail.
//
// The throttle stamp is per-project (like the tick's), so a run in one project
// doesn't starve another's project-scoped teams for 24h; the detached
// `atl update` inherits session-start's cwd, updating that project plus the
// shared global layer. `ATL_NO_TEAM_UPDATE` opts out.
func autoUpdateTeams(project string) {
	if os.Getenv(envNoTeamUpdate) != "" {
		return
	}
	stamp, err := throttle.StampPath("last-team-update-" + projectStamp(project))
	if err != nil {
		return
	}
	if !throttle.Gate(stamp, teamUpdateInterval) {
		return // checked recently
	}
	_ = throttle.Touch(stamp)
	_ = teamUpdateSpawn() // best-effort; the next session sees the result
}
