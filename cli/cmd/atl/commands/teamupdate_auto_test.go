package commands

import (
	"os"
	"testing"

	"github.com/agentteamland/atl/cli/internal/throttle"
)

// autoUpdateTeams must gate the detached spawn on both the env brake and the
// per-project throttle. The spawn itself is injected so the test never forks a
// real `atl update`.
func TestAutoUpdateTeams(t *testing.T) {
	// Isolate the throttle stamp under a temp HOME so the real ~/.atl/cache is
	// untouched (throttle.StampPath resolves via os.UserHomeDir → $HOME on unix).
	t.Setenv("HOME", t.TempDir())

	var spawns int
	orig := teamUpdateSpawn
	teamUpdateSpawn = func() error { spawns++; return nil }
	t.Cleanup(func() { teamUpdateSpawn = orig })

	const project = "/tmp/some/project"

	// Env brake: no spawn.
	t.Setenv(envNoTeamUpdate, "1")
	autoUpdateTeams(project)
	if spawns != 0 {
		t.Fatalf("env brake should prevent spawn, got %d", spawns)
	}
	if err := os.Unsetenv(envNoTeamUpdate); err != nil {
		t.Fatal(err)
	}

	// First run (no stamp yet): spawns once and records the stamp.
	autoUpdateTeams(project)
	if spawns != 1 {
		t.Fatalf("first run should spawn once, got %d", spawns)
	}
	stamp, err := throttle.StampPath("last-team-update-" + projectStamp(project))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(stamp); err != nil {
		t.Fatalf("stamp should exist after a run: %v", err)
	}

	// Throttled: an immediate second run within the interval does not spawn again.
	autoUpdateTeams(project)
	if spawns != 1 {
		t.Fatalf("throttled run should not spawn, got %d", spawns)
	}

	// A different project has its own stamp → not throttled by the first project.
	autoUpdateTeams("/tmp/other/project")
	if spawns != 2 {
		t.Fatalf("a different project should spawn (own throttle), got %d", spawns)
	}
}
