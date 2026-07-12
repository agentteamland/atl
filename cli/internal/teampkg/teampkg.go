// Package teampkg reads a fetched team's manifest and reflects its assets into
// Claude Code's directory — the "write" half of install.
//
// v2 keeps a single copy: assets go straight into the scope's .claude dir
// (~/.claude or <project>/.claude), not a parallel ATL-owned store (decision
// 2026-06-14, asset model (b)). agents/skills/rules are the directories Claude
// Code reads directly; knowledge/scripts carry a team's runtime reference docs +
// helper scripts (consulted by its agents/skills/workers at run time) — team.json
// and repo chrome (README, LICENSE) stay behind. Each copied file's SHA-256 is
// recorded into the returned files map, which becomes the install manifest's
// fanout baseline + integrity set.
package teampkg

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/agentteamland/atl/cli/internal/fanout"
)

// AssetDirs are the top-level subtrees reflected into .claude, and the single
// source of truth for "what counts as an installable asset" — shared by install
// (CopyAssets), update (reflectWithFanout), and reclamation (gc.Scan) so the three
// can never drift. agents/skills/rules are what Claude Code reads directly;
// knowledge/scripts carry a team's runtime reference docs + helper scripts (e.g.
// delivery-team's Azure adapter contract + attachment helper); packs carry a
// software team's area-keyed stack knowledge — the M1 seam, where a generic
// developer worker loads packs/<area>/ for the work-unit's tech-lead-tagged area.
// All are consulted by a team's agents/skills/workers at run time. Everything else
// in the team repo (team.json, README, LICENSE, ...) is not an installable asset.
var AssetDirs = []string{"agents", "skills", "rules", "knowledge", "scripts", "packs"}

// TeamManifest is the subset of team.json install needs. Extra v1 fields
// (agents[], capabilities, extends, ...) are tolerated and ignored.
type TeamManifest struct {
	SchemaVersion int    `json:"schemaVersion"`
	Name          string `json:"name"`
	Version       string `json:"version"`
	Description   string `json:"description"`
	Scope         string `json:"scope"` // v2 addition; "" = project (see internal/scope.Parse)
	// Dependencies maps a team name (or "<handle>/<name>") to a semver range.
	// "core" is the platform core (always present) and is skipped by install;
	// every other entry is resolved from the index and installed transitively.
	Dependencies map[string]string `json:"dependencies"`
}

// ReadManifest loads and minimally validates team.json from a fetched team dir.
func ReadManifest(dir string) (*TeamManifest, error) {
	b, err := os.ReadFile(filepath.Join(dir, "team.json"))
	if err != nil {
		return nil, fmt.Errorf("read team.json: %w", err)
	}
	var tm TeamManifest
	if err := json.Unmarshal(b, &tm); err != nil {
		return nil, fmt.Errorf("parse team.json: %w", err)
	}
	if tm.Name == "" {
		return nil, fmt.Errorf("team.json has no name")
	}
	return &tm, nil
}

// CopyAssets reflects srcDir's asset subtrees (see AssetDirs) into claudeDir and
// returns a map of each written file's path (relative to claudeDir,
// slash-separated) to its SHA-256. Errors if the team ships no assets.
func CopyAssets(srcDir, claudeDir string) (map[string]string, error) {
	files := map[string]string{}
	for _, ad := range AssetDirs {
		srcAd := filepath.Join(srcDir, ad)
		info, err := os.Stat(srcAd)
		if err != nil || !info.IsDir() {
			continue // this team doesn't ship that asset kind
		}
		walkErr := filepath.WalkDir(srcAd, func(p string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			rel, err := filepath.Rel(srcDir, p) // e.g. agents/api/agent.md
			if err != nil {
				return err
			}
			dst := filepath.Join(claudeDir, rel)
			if err := CopyFile(p, dst); err != nil {
				return err
			}
			h, err := fanout.HashFile(dst)
			if err != nil {
				return err
			}
			files[filepath.ToSlash(rel)] = h
			return nil
		})
		if walkErr != nil {
			return nil, walkErr
		}
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("team ships no installable assets (%s)", strings.Join(AssetDirs, "/"))
	}
	return files, nil
}

// CopyFile copies src to dst, creating parent directories as needed. Shared by
// install (asset reflect) and update (fan-out refresh).
func CopyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	b, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	if err := os.WriteFile(dst, b, info.Mode().Perm()); err != nil {
		return err
	}
	// WriteFile's perm only applies when creating; on a fan-out / integrity
	// overwrite the file already exists, so chmod explicitly to preserve the
	// source mode — an executable helper script (scripts/*.sh) must stay +x when
	// reflected, not silently drop to 0644.
	return os.Chmod(dst, info.Mode().Perm())
}
