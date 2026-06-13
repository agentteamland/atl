package commands

import (
	"fmt"

	"github.com/agentteamland/atl/cli/internal/doctor"
	"github.com/agentteamland/atl/cli/internal/integrity"
	"github.com/agentteamland/atl/cli/internal/scope"
)

// integrityCheck is the doctor check that restores missing installed files
// (#2b). It scans both layers, restores absent files from their pinned source,
// and reports — Healed when it actually restored something (visible, never
// silent). A restore failure is a warning, not a session-blocker.
func integrityCheck(projectRoot string) doctor.Check {
	return func() doctor.Result {
		restored := 0
		var firstErr error
		for _, s := range []scope.Scope{scope.Project, scope.Global} {
			n, err := healScope(s, projectRoot)
			restored += n
			if err != nil && firstErr == nil {
				firstErr = err
			}
		}
		if firstErr != nil {
			return doctor.Result{Name: "asset-integrity", Status: doctor.Warn,
				Detail: "could not restore a missing file: " + firstErr.Error()}
		}
		if restored > 0 {
			return doctor.Result{Name: "asset-integrity", Status: doctor.OK, Healed: true,
				Detail: fmt.Sprintf("restored %d missing file(s) — `atl remove <handle>/<team>` removes a team for good", restored)}
		}
		return doctor.Result{Name: "asset-integrity", Status: doctor.OK, Detail: "all installed files present"}
	}
}

// healScope scans one layer for missing installed files and restores them.
func healScope(s scope.Scope, projectRoot string) (int, error) {
	layer, err := scope.LayerDir(s, projectRoot)
	if err != nil {
		return 0, err
	}
	claude, err := scope.ClaudeDir(s, projectRoot)
	if err != nil {
		return 0, err
	}
	missing, err := integrity.Scan(layer, claude)
	if err != nil {
		return 0, err
	}
	if len(missing) == 0 {
		return 0, nil
	}
	return integrity.RestoreAll(missing, claude)
}
