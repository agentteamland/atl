// Package gc reclaims orphaned assets — the missing inverse of install.
//
// install/update/promote write agents, skills, and rules into ~/.claude and
// <proj>/.claude and record each in an install manifest. Nothing prunes what
// falls out of that contract: a file dropped upstream on update (left on disk
// by design — "a future prune can remove"), a learning-loop gain left behind
// after a team is removed, or a file a user dropped in by hand. gc finds those
// zero-owner files and reclaims them *reversibly* — a dry-run report by default,
// a soft-delete into ~/.atl/gc-trash on --apply, restorable via --undo.
//
// doctor HEALS (restores manifest-listed files that went absent); gc PRUNES
// (removes files no manifest owns). They are deliberate opposites: doctor never
// deletes, gc never deletes irreversibly (soft-delete + undo + explicit purge).
package gc

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/agentteamland/atl/cli/internal/coreassets"
	"github.com/agentteamland/atl/cli/internal/manifest"
	"github.com/agentteamland/atl/cli/internal/pin"
	"github.com/agentteamland/atl/cli/internal/scope"
	"github.com/agentteamland/atl/cli/internal/teampkg"
)

// HistoryMaxAge is how long a promote conflict-archive survives before gc treats
// it as reclaimable. Conflict archives are content-addressed and never pruned by
// anything else, so they accumulate forever without this.
const HistoryMaxAge = 30 * 24 * time.Hour

// Orphan is one reclaimable item: an on-disk asset file no manifest at its layer
// owns, or an expired conflict archive.
type Orphan struct {
	Scope string // "global" | "project" | "history"
	Rel   string // path relative to the .claude dir (or, for history, <handle>__<name>/<hash>)
	Abs   string // absolute path on disk
	Unit  string // the owning asset unit (e.g. "agents/foo"), for the origin hint
	Owned bool   // Unit is owned by some manifest at this layer (a sibling gain vs a wholly-unowned dir)
	Size  int64
}

// Origin is a human hint for why this item is reclaimable — never a certainty.
func (o Orphan) Origin() string {
	switch {
	case o.Scope == "history":
		return "expired conflict archive"
	case o.Owned:
		return "gain or edit beside an installed unit"
	default:
		return "unowned unit (a removed team or a hand-made dir)"
	}
}

// Scan returns every reclaimable item: zero-owner asset files across the global
// and project layers, plus conflict archives older than HistoryMaxAge. `now` is
// injected for testability. Sorted by scope then path.
func Scan(projectRoot string, now time.Time) ([]Orphan, error) {
	// Core rules + skills are reflected into ~/.claude from the binary with no
	// install manifest — they are platform assets, not orphans. Treat them as
	// owned at the global layer so gc never flags (or deletes) them.
	corePaths, err := coreassets.Paths()
	if err != nil {
		return nil, err
	}
	core := map[string]bool{}
	for _, p := range corePaths {
		core[p] = true
	}

	var out []Orphan
	for _, sc := range []scope.Scope{scope.Global, scope.Project} {
		layerDir, err := scope.LayerDir(sc, projectRoot)
		if err != nil {
			return nil, err
		}
		claudeDir, err := scope.ClaudeDir(sc, projectRoot)
		if err != nil {
			return nil, err
		}
		var extraOwned map[string]bool
		var pinned func(string) bool
		if sc == scope.Global {
			extraOwned = core
		} else {
			// A project pin is an explicit "keep this project-only" — a stronger
			// ownership signal than a manifest entry, so gc must never flag (or
			// sweep) a pinned path or its subtree.
			if pins, perr := pin.Load(layerDir); perr == nil {
				pinned = pins.Pinned
			}
		}
		orphans, err := scanLayer(sc.String(), layerDir, claudeDir, extraOwned, pinned)
		if err != nil {
			return nil, err
		}
		out = append(out, orphans...)
	}
	hist, err := scanHistory(now)
	if err != nil {
		return nil, err
	}
	out = append(out, hist...)

	sort.Slice(out, func(i, j int) bool {
		if out[i].Scope != out[j].Scope {
			return out[i].Scope < out[j].Scope
		}
		return out[i].Rel < out[j].Rel
	})
	return out, nil
}

// scanLayer walks a layer's asset dirs and returns files no manifest at that
// layer claims. extraOwned holds paths owned outside any manifest (core assets at
// the global layer). pinned, when non-nil, reports project-pinned paths to treat
// as owned. A missing asset dir (the layer has no installs) yields nothing.
func scanLayer(scopeName, layerDir, claudeDir string, extraOwned map[string]bool, pinned func(string) bool) ([]Orphan, error) {
	manifests, err := manifest.List(layerDir)
	if err != nil {
		return nil, err
	}
	owned := map[string]bool{}      // exact rel paths a manifest (or core) claims
	ownedUnits := map[string]bool{} // "agents/foo" prefixes owned by a manifest or core
	for rel := range extraOwned {
		owned[rel] = true
		if u := unitOf(rel); u != "" {
			ownedUnits[u] = true
		}
	}
	for _, m := range manifests {
		for rel := range m.Files {
			owned[rel] = true
			if u := unitOf(rel); u != "" {
				ownedUnits[u] = true
			}
		}
	}

	var out []Orphan
	for _, dir := range teampkg.AssetDirs {
		root := filepath.Join(claudeDir, dir)
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				if os.IsNotExist(walkErr) {
					return nil // this asset dir doesn't exist at this layer
				}
				return walkErr
			}
			if d.IsDir() {
				return nil
			}
			rel, rerr := filepath.Rel(claudeDir, path)
			if rerr != nil {
				return rerr
			}
			rel = filepath.ToSlash(rel)
			if owned[rel] {
				return nil // a manifest owns this file
			}
			if pinned != nil && pinned(rel) {
				return nil // pinned project-only → treated as owned, never reclaimed
			}
			info, ierr := d.Info()
			if ierr != nil {
				return ierr
			}
			u := unitOf(rel)
			out = append(out, Orphan{
				Scope: scopeName, Rel: rel, Abs: path,
				Unit: u, Owned: ownedUnits[u], Size: info.Size(),
			})
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

// scanHistory returns promote conflict-archive snapshots older than HistoryMaxAge.
// Layout: ~/.atl/history/<handle>__<name>/<hash>/<rel...>. The prune unit is the
// <hash> snapshot dir.
func scanHistory(now time.Time) ([]Orphan, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	histRoot := filepath.Join(home, ".atl", "history")
	teams, err := os.ReadDir(histRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []Orphan
	for _, t := range teams {
		if !t.IsDir() {
			continue
		}
		teamDir := filepath.Join(histRoot, t.Name())
		snaps, err := os.ReadDir(teamDir)
		if err != nil {
			continue
		}
		for _, s := range snaps {
			if !s.IsDir() {
				continue
			}
			info, ierr := s.Info()
			if ierr != nil {
				continue
			}
			if now.Sub(info.ModTime()) <= HistoryMaxAge {
				continue
			}
			abs := filepath.Join(teamDir, s.Name())
			out = append(out, Orphan{
				Scope: "history",
				Rel:   filepath.ToSlash(filepath.Join(t.Name(), s.Name())),
				Abs:   abs, Unit: t.Name(), Size: dirSize(abs),
			})
		}
	}
	return out, nil
}

// unitOf returns the "agents/foo" owning unit (first two path segments).
func unitOf(rel string) string {
	parts := strings.Split(rel, "/")
	if len(parts) >= 2 {
		return parts[0] + "/" + parts[1]
	}
	return ""
}

func dirSize(root string) int64 {
	var total int64
	_ = filepath.WalkDir(root, func(_ string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if info, ierr := d.Info(); ierr == nil {
			total += info.Size()
		}
		return nil
	})
	return total
}

// TrashRoot is ~/.atl/gc-trash, where soft-deleted items wait for expiry or undo.
func TrashRoot() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".atl", "gc-trash"), nil
}

// undoEntry records one moved item so Undo can restore it to its original place.
type undoEntry struct {
	Orig  string `json:"orig"`
	Trash string `json:"trash"`
}

// SoftDelete moves each orphan into a fresh batch under trashRoot (named `stamp`,
// injected for testability) and writes an undo manifest. Nothing is hard-deleted.
// Returns the batch dir.
func SoftDelete(orphans []Orphan, trashRoot, stamp string) (string, error) {
	batchDir := filepath.Join(trashRoot, stamp)
	if err := os.MkdirAll(filepath.Join(batchDir, "files"), 0o755); err != nil {
		return "", err
	}
	var entries []undoEntry
	for _, o := range orphans {
		trashPath := filepath.Join(batchDir, "files", o.Scope, filepath.FromSlash(o.Rel))
		if err := moveEntry(o.Abs, trashPath); err != nil {
			return "", err
		}
		entries = append(entries, undoEntry{Orig: o.Abs, Trash: trashPath})
	}
	b, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(batchDir, "undo.json"), append(b, '\n'), 0o644); err != nil {
		return "", err
	}
	return batchDir, nil
}

// Undo restores the most recent soft-delete batch back to original paths and
// removes the batch. Returns how many items were restored (0 if trash is empty).
func Undo(trashRoot string) (int, error) {
	batch, err := latestBatch(trashRoot)
	if err != nil || batch == "" {
		return 0, err
	}
	b, err := os.ReadFile(filepath.Join(batch, "undo.json"))
	if err != nil {
		return 0, err
	}
	var entries []undoEntry
	if err := json.Unmarshal(b, &entries); err != nil {
		return 0, err
	}
	restored := 0
	for _, e := range entries {
		if err := moveEntry(e.Trash, e.Orig); err != nil {
			return restored, err
		}
		restored++
	}
	if err := os.RemoveAll(batch); err != nil {
		return restored, err
	}
	return restored, nil
}

// Purge hard-deletes trash batches older than olderThan (or every batch when
// olderThan is 0). This is the only place gc deletes for real. Returns the count.
func Purge(trashRoot string, olderThan time.Duration, now time.Time) (int, error) {
	entries, err := os.ReadDir(trashRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	purged := 0
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		info, ierr := e.Info()
		if ierr != nil {
			continue
		}
		if olderThan != 0 && now.Sub(info.ModTime()) <= olderThan {
			continue
		}
		if err := os.RemoveAll(filepath.Join(trashRoot, e.Name())); err != nil {
			return purged, err
		}
		purged++
	}
	return purged, nil
}

// latestBatch returns the newest batch dir under trashRoot (lexically last —
// batch names are sortable timestamps), or "" if there are none.
func latestBatch(trashRoot string) (string, error) {
	entries, err := os.ReadDir(trashRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	if len(names) == 0 {
		return "", nil
	}
	sort.Strings(names)
	return filepath.Join(trashRoot, names[len(names)-1]), nil
}

// moveEntry moves src to dst (rename, with a copy+remove fallback for a file that
// crosses a device boundary — a project-layer file trashed into ~/.atl). History
// archives live under ~/.atl already, so their directory moves never cross a
// device and never hit the fallback.
func moveEntry(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	if err := os.Rename(src, dst); err == nil {
		return nil
	}
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("gc: cannot move directory across devices: %s", src)
	}
	if err := teampkg.CopyFile(src, dst); err != nil {
		return err
	}
	return os.Remove(src)
}
