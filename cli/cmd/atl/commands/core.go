package commands

import (
	"strings"

	"github.com/agentteamland/atl/cli/internal/coreassets"
	"github.com/agentteamland/atl/cli/internal/manifest"
	"github.com/agentteamland/atl/cli/internal/scope"
	"github.com/agentteamland/atl/cli/internal/userrules"
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

// reflectUserRules reflects a scope's user-authored rules — the ones /rule writes
// to <layer>/.atl/rules — into that scope's Claude Code load surface
// (<layer>/.claude/rules), so a rule authored via `/rule` actually loads.
// Platform- and team-owned rule names are protected (see ownedRuleNames): a user
// rule that collides with a core or team-installed rule name simply isn't
// reflected, so installed content is never silently overwritten. Returns the
// count of files that actually changed.
func reflectUserRules(sc scope.Scope, projectRoot string) (int, error) {
	layerDir, err := scope.LayerDir(sc, projectRoot)
	if err != nil {
		return 0, err
	}
	claudeDir, err := scope.ClaudeDir(sc, projectRoot)
	if err != nil {
		return 0, err
	}
	protected, err := ownedRuleNames(sc, layerDir)
	if err != nil {
		return 0, err
	}
	return userrules.Reflect(layerDir, claudeDir, protected)
}

// ownedRuleNames returns the rule basenames already owned at a layer — the core
// rules (global only, reflected from the binary) plus any rules a team installed
// via its manifest — so a user rule never silently overwrites platform or
// team-installed content. Core wins at global; an installed team owns its rule
// slot at whichever layer it was installed into.
func ownedRuleNames(sc scope.Scope, layerDir string) (map[string]bool, error) {
	out := map[string]bool{}
	if sc == scope.Global {
		paths, err := coreassets.Paths()
		if err != nil {
			return nil, err
		}
		for _, p := range paths {
			if name, ok := strings.CutPrefix(p, "rules/"); ok {
				out[name] = true
			}
		}
	}
	manifests, err := manifest.List(layerDir)
	if err != nil {
		return nil, err
	}
	for _, m := range manifests {
		for rel := range m.Files {
			if name, ok := strings.CutPrefix(rel, "rules/"); ok {
				out[name] = true
			}
		}
	}
	return out, nil
}
