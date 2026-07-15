package commands

import (
	"fmt"
	"strings"

	"github.com/agentteamland/atl/cli/internal/coreassets"
	"github.com/agentteamland/atl/cli/internal/manifest"
	"github.com/agentteamland/atl/cli/internal/scope"
	"github.com/agentteamland/atl/cli/internal/userrules"
)

// autoDrainNotice is the per-turn signal that tells the agent to drain the
// learning queue in the background — empty when the queue is empty. Both the
// UserPromptSubmit tick and session-start emit it; the learning-capture rule is
// what acts on it (spawn a background drain subagent).
func autoDrainNotice(pending int) string {
	return drainSignal(pending, "learning", "learning-capture")
}

// autoProfileDrainNotice is the profile-fact sibling of autoDrainNotice: the
// per-turn signal telling the agent to fold pending profile-fact markers into
// the global profiles in the background via /profile-drain. The action
// instruction lives in profile-team's profile-capture rule (core stays
// team-agnostic), so when profile-team isn't installed no rule acts on this and
// it is a harmless notice.
func autoProfileDrainNotice(pending int) string {
	return drainSignal(pending, "profile-fact", "profile-capture")
}

// drainSignal builds the shared auto-drain signal for one queue channel — a
// single wording template for every channel, so the two emitters (tick +
// session-start) and the two channels can't drift. noun is the singular item
// name ("learning", "profile-fact"); rule is the capture rule that carries the
// action instruction. Empty when the channel is empty (no false-fire).
func drainSignal(pending int, noun, rule string) string {
	if pending <= 0 {
		return ""
	}
	return fmt.Sprintf("atl: %d %s(s) pending — auto-drain them now in a background subagent (per the %s rule)", pending, noun, rule)
}

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
