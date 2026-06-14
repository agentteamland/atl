// Package promote computes how a project's local gains are lifted into the
// user-global layer — the upward mirror of internal/fanout.
//
// Fan-out pulls global→project, refreshing files the project never touched and
// preserving the ones it did. Promote does the inverse: it lifts the files the
// project *did* evolve past its install baseline (overwhelmingly the learning
// loop's output — an agent's grown children/ and rebuilt knowledge base) up into
// the global copy, so the user's other projects fan them out next.
//
// This is the pure planner: it hashes files and decides, but performs no writes.
// The command applies the plan (copy, archive-on-conflict, advance baselines).
package promote

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/agentteamland/atl/cli/internal/fanout"
	"github.com/agentteamland/atl/cli/internal/manifest"
)

// Kind classifies one lift.
type Kind int

const (
	// Lift: the project evolved this file and global is still at the shared
	// baseline (or doesn't have it yet) — a clean upward copy.
	Lift Kind = iota
	// ConflictLift: both project and global moved this file independently. The
	// project value wins; the prior global value is archived first.
	ConflictLift
)

func (k Kind) String() string {
	if k == ConflictLift {
		return "conflict-lift"
	}
	return "lift"
}

// Action is one file to lift from project to global.
type Action struct {
	Rel           string // path relative to the .claude dir (slash-separated)
	Kind          Kind
	ProjHash      string // the project hash being promoted (becomes the new baseline)
	PriorGlobHash string // global's pre-lift hash ("" if absent) — archived on conflict
}

// Plan computes the lifts for one team installed at both scopes. pm is the
// project manifest (its Files map is the install baseline); projClaude and
// globClaude are the two .claude roots; pinned reports project-only opt-outs.
// It writes nothing.
func Plan(pm *manifest.Manifest, projClaude, globClaude string, pinned func(string) bool) ([]Action, error) {
	var actions []Action
	tracked := make(map[string]bool, len(pm.Files))

	// 1. Modified tracked files — a file the project changed past its baseline.
	for rel, baseline := range pm.Files {
		tracked[rel] = true
		if pinned(rel) {
			continue
		}
		proj, err := fanout.HashFile(filepath.Join(projClaude, filepath.FromSlash(rel)))
		if err != nil {
			return nil, err
		}
		if proj == "" {
			continue // deleted locally — doctor's integrity-restore lane, not promote's
		}
		glob, err := fanout.HashFile(filepath.Join(globClaude, filepath.FromSlash(rel)))
		if err != nil {
			return nil, err
		}
		if proj == glob || proj == baseline {
			continue // already shared, or the project never evolved it
		}
		k := Lift
		if glob != baseline {
			k = ConflictLift // global moved too
		}
		actions = append(actions, Action{Rel: rel, Kind: k, ProjHash: proj, PriorGlobHash: glob})
	}

	// 2. New files under the team's owned units (e.g. an agent's grown
	//    children/) that aren't in the manifest yet — additive gains.
	for _, unit := range ownedUnits(pm.Files) {
		root := filepath.Join(projClaude, filepath.FromSlash(unit))
		info, err := os.Stat(root)
		if err != nil || !info.IsDir() {
			continue // a flat file unit (e.g. rules/x.md) has no children to scan
		}
		walkErr := filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}
			rel, err := filepath.Rel(projClaude, p)
			if err != nil {
				return err
			}
			relSlash := filepath.ToSlash(rel)
			if tracked[relSlash] || pinned(relSlash) {
				return nil
			}
			proj, err := fanout.HashFile(p)
			if err != nil {
				return err
			}
			glob, err := fanout.HashFile(filepath.Join(globClaude, filepath.FromSlash(relSlash)))
			if err != nil {
				return err
			}
			if proj == glob {
				return nil // already identical in global
			}
			k := Lift
			if glob != "" {
				k = ConflictLift
			}
			actions = append(actions, Action{Rel: relSlash, Kind: k, ProjHash: proj, PriorGlobHash: glob})
			return nil
		})
		if walkErr != nil {
			return nil, walkErr
		}
	}

	sort.Slice(actions, func(i, j int) bool { return actions[i].Rel < actions[j].Rel })
	return actions, nil
}

// ownedUnits derives the asset units a team owns from its manifest files — the
// first two path segments (agents/<x>, skills/<x>, rules/<x>) — so new-file
// discovery scans only this team's subtrees, never a sibling team's.
func ownedUnits(files map[string]string) []string {
	seen := map[string]bool{}
	var units []string
	for rel := range files {
		parts := strings.SplitN(rel, "/", 3)
		unit := rel
		if len(parts) >= 2 {
			unit = parts[0] + "/" + parts[1]
		}
		if !seen[unit] {
			seen[unit] = true
			units = append(units, unit)
		}
	}
	sort.Strings(units)
	return units
}
