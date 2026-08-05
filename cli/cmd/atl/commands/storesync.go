package commands

import (
	"fmt"

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
// became recoverable), while the tick stays silent, since it runs once per
// throttle window and a recurring notice would be noise rather than signal.
func versionDeclaredStores(projectRoot string) int {
	return storegit.EnsureAll(declaredStores(projectRoot))
}

// declaredStores collects every store path the installed teams declared, across
// both scopes. Shared by the versioning pass and the off-machine report so the
// two can never disagree about which stores exist.
func declaredStores(projectRoot string) []string {
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
	return dirs
}

// reportUnbackedStores says, once per session, when a declared store's history
// exists nowhere but this disk.
//
// Local versioning and an off-machine copy are different jobs: EnsureAll makes an
// overwritten value recoverable and does nothing about the disk failing. Only the
// first had a mechanism, so a store could accumulate months of irreplaceable
// content while every automatic pass reported success and nothing ever said the
// copy had not left the machine.
//
// Deliberately unthrottled. This is not an event that happened once — it is a
// standing unsafe state that stays true until the user acts, and a stamp would
// mean exactly one project per window ever hears about it. It goes silent the
// moment it is fixed, which is what keeps it from becoming wallpaper.
//
// Core names no team here. The condition and its consequence are all this can
// honestly say; which skill attaches a remote belongs to the rule the owning
// team ships, reflected into the agent's always-loaded rules on install.
func reportUnbackedStores(projectRoot string) {
	for _, st := range storegit.Backups(declaredStores(projectRoot)) {
		where := storegit.Display(st.Dir)
		switch {
		case !st.HasRemote:
			fmt.Printf("atl: the durable store at %s has no remote — its history exists only on this disk, "+
				"so losing the disk loses it. Raise this with the user in this session and follow the backup "+
				"rule for that store's owning team.\n", where)
		case st.Unpushed > 0:
			fmt.Printf("atl: the durable store at %s is %d commit(s) ahead of its remote — that much of it "+
				"exists only on this disk. Raise this with the user in this session and follow the backup "+
				"rule for that store's owning team.\n", where, st.Unpushed)
		}
	}
}
