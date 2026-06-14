package commands

import (
	"github.com/agentteamland/atl/cli/internal/coreassets"
	"github.com/agentteamland/atl/cli/internal/scope"
)

// reflectCore refreshes the embedded core rules + skills into the global Claude
// dir. Core is the platform layer — it ships inside the binary and is reflected
// on install, update, and session start so it's always present and in lockstep
// with the binary version. Returns the count of files that actually changed.
func reflectCore() (int, error) {
	globalClaude, err := scope.ClaudeDir(scope.Global, "")
	if err != nil {
		return 0, err
	}
	return coreassets.Reflect(globalClaude)
}
