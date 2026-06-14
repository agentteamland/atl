package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/agentteamland/atl/cli/internal/generation"
	"github.com/agentteamland/atl/cli/internal/manifest"
	"github.com/agentteamland/atl/cli/internal/pin"
	"github.com/agentteamland/atl/cli/internal/promote"
	"github.com/agentteamland/atl/cli/internal/scope"
	"github.com/agentteamland/atl/cli/internal/teampkg"
	"github.com/spf13/cobra"
)

var promoteCmd = &cobra.Command{
	Use:   "promote [handle/team]",
	Short: "Lift project-local gains to the user-global layer",
	Long: "Lift the gains your project's agents have accumulated (their grown\n" +
		"knowledge base) into the global-layer copy of their team — ring 1→2 of\n" +
		"gain circulation, the upward mirror of update's fan-out. It runs\n" +
		"automatically in the background tick; this is the manual surface.\n\n" +
		"Only teams installed at both project and global scope are promoted (there\n" +
		"must be a global copy to lift into). Additive and safe: a file the global\n" +
		"layer also changed is a true conflict — the project value wins and the\n" +
		"prior global value is archived under ~/.atl/history. Files you `atl pin`\n" +
		"are kept project-only and never lifted.",
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		projectRoot, err := os.Getwd()
		if err != nil {
			return err
		}
		var only string
		if len(args) == 1 {
			only = args[0]
		}
		res, err := promoteGains(projectRoot, only)
		if err != nil {
			return err
		}
		fmt.Println(res.String())
		return nil
	},
}

// promoteResult tallies one promote pass.
type promoteResult struct {
	lifted    int
	conflicts int
	teams     int
}

func (r promoteResult) String() string {
	if r.lifted == 0 {
		return "atl promote: nothing to lift — the global layer is already current"
	}
	s := fmt.Sprintf("atl promote: lifted %d file(s) to the global layer", r.lifted)
	if r.conflicts > 0 {
		s += fmt.Sprintf(" (%d conflict(s) — project won, prior global archived to ~/.atl/history)", r.conflicts)
	}
	return s
}

// promoteGains lifts every eligible project team's gains into the global layer.
// only, if non-empty ("handle/name"), restricts the pass to that one team.
// Shared by the manual command and the automatic tick pass.
//
// For each team installed at BOTH scopes, promote.Plan decides which files the
// project evolved; each is copied project→global, the prior global value is
// archived on a true conflict, and both manifests' baselines advance to the
// promoted hash (so the same gain is never lifted twice and a later fan-out
// treats it as up-to-date). A successful pass bumps the global generation so the
// user's other projects fan the gains out on their next tick — closing the ring.
func promoteGains(projectRoot, only string) (promoteResult, error) {
	var res promoteResult

	projLayer, err := scope.LayerDir(scope.Project, projectRoot)
	if err != nil {
		return res, err
	}
	globLayer, err := scope.LayerDir(scope.Global, "")
	if err != nil {
		return res, err
	}
	projClaude, err := scope.ClaudeDir(scope.Project, projectRoot)
	if err != nil {
		return res, err
	}
	globClaude, err := scope.ClaudeDir(scope.Global, "")
	if err != nil {
		return res, err
	}

	pins, err := pin.Load(projLayer)
	if err != nil {
		return res, err
	}
	projManifests, err := manifest.List(projLayer)
	if err != nil {
		return res, err
	}

	bumped := false
	for _, pm := range projManifests {
		if only != "" && pm.Handle+"/"+pm.Name != only {
			continue
		}
		gm, gerr := manifest.Read(globLayer, pm.Handle, pm.Name)
		if gerr != nil {
			continue // not installed globally → no upstream copy to lift into
		}
		actions, perr := promote.Plan(pm, projClaude, globClaude, pins.Pinned)
		if perr != nil {
			return res, perr
		}
		if len(actions) == 0 {
			continue
		}
		if gm.Files == nil {
			gm.Files = map[string]string{}
		}
		for _, a := range actions {
			if a.Kind == promote.ConflictLift {
				if err := archivePriorGlobal(globLayer, globClaude, pm.Handle, pm.Name, a.Rel, a.PriorGlobHash); err != nil {
					return res, err
				}
				res.conflicts++
			}
			src := filepath.Join(projClaude, filepath.FromSlash(a.Rel))
			dst := filepath.Join(globClaude, filepath.FromSlash(a.Rel))
			if err := teampkg.CopyFile(src, dst); err != nil {
				return res, err
			}
			gm.Files[a.Rel] = a.ProjHash // global now holds the gain
			pm.Files[a.Rel] = a.ProjHash // project baseline advances — no re-promote
			res.lifted++
		}
		if err := gm.Write(globLayer); err != nil {
			return res, err
		}
		if err := pm.Write(projLayer); err != nil {
			return res, err
		}
		res.teams++
		bumped = true
	}
	if bumped {
		_ = generation.Bump() // global layer changed → other projects fan out next tick
	}
	return res, nil
}

// archivePriorGlobal saves the global file's current bytes under
// <globLayer>/history/<handle>__<name>/<priorHash>/<rel> before a conflict lift
// overwrites it — the reversible substrate. Content-addressed, so the same prior
// value is never archived twice.
func archivePriorGlobal(globLayer, globClaude, handle, name, rel, priorHash string) error {
	if priorHash == "" {
		return nil // nothing there to archive
	}
	dst := filepath.Join(globLayer, "history", handle+"__"+name, priorHash, filepath.FromSlash(rel))
	if _, err := os.Stat(dst); err == nil {
		return nil // already archived (dedup)
	}
	return teampkg.CopyFile(filepath.Join(globClaude, filepath.FromSlash(rel)), dst)
}
