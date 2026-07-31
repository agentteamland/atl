package commands

import (
	"github.com/agentteamland/atl/cli/internal/manifest"
	"github.com/agentteamland/atl/cli/internal/scope"
	"github.com/agentteamland/atl/cli/internal/storegit"
)

// versionDeclaredStores keeps every durable store an installed team DECLARED
// under local version control, committing whatever changed since the last
// session.
//
// The stores come from the install manifests (`stores`, recorded at install and
// refreshed on update from the team's own `capabilities.<name>.store`), so core
// never learns which team owns which path — it honors a declaration. Reading a
// declared path for a mechanical purpose is not enforcing an access contract;
// who may read or write a store is a separate, still-deferred question.
//
// Why in Go rather than in the owning team's skill: a store is written from
// several directions — a drain subagent, a team agent writing directly, the user
// editing a file by hand — and only a deterministic pass at session boundaries
// catches all of them. An instruction to "remember to commit" catches whichever
// path happens to be reading that instruction at the time.
//
// Never fails. Returns how many stores produced a commit, so the caller decides
// whether to say anything: session-start reports it (a real event — values just
// became recoverable), the per-turn tick stays silent (it runs far too often for
// a notice to be signal).
func versionDeclaredStores(projectRoot string) int {
	layers := []scope.Scope{scope.Global}
	if projectRoot != "" {
		// An unresolved project root would make LayerDir return a RELATIVE ".atl",
		// silently reading whatever happens to sit under the current directory.
		layers = append(layers, scope.Project)
	}
	var dirs []string
	for _, s := range layers {
		layer, err := scope.LayerDir(s, projectRoot)
		if err != nil {
			continue
		}
		manifests, err := manifest.List(layer)
		if err != nil {
			continue
		}
		for _, m := range manifests {
			dirs = append(dirs, m.Stores...)
		}
	}
	return storegit.EnsureAll(dirs)
}
