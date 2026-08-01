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
	"sort"
	"strings"

	"github.com/agentteamland/atl/cli/internal/fanout"
	"github.com/agentteamland/atl/cli/internal/manifest"
)

// AssetDirs are the top-level subtrees reflected into .claude, and the single
// source of truth for "what counts as an installable asset" — shared by install
// (CopyAssets), update (reflectWithFanout), and reclamation (gc.Scan) so the three
// can never drift. agents/skills/rules are what Claude Code reads directly;
// knowledge/scripts carry a team's runtime reference docs + helper scripts (e.g.
// delivery-team's attachment helper); backends carry a team's per-backend adapter
// contracts (delivery-team's backends/{azure,github}/adapter.md — the operation→tool
// map every ceremony + worker loads for the active backend); packs carry a software
// team's area-keyed stack knowledge — the M1 seam, where a generic developer worker
// loads packs/<area>/ for the work-unit's tech-lead-tagged area.
// All are consulted by a team's agents/skills/workers at run time. Everything else
// in the team repo (team.json, README, LICENSE, ...) is not an installable asset.
var AssetDirs = []string{"agents", "skills", "rules", "knowledge", "backends", "scripts", "packs"}

// TeamManifest is the subset of team.json install needs. Extra v1 fields
// (agents[], extends, ...) are tolerated and ignored.
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
	// Capabilities is read ONLY to learn where a team keeps a durable store it
	// wants versioned (see DeclaredStores) and which capture channels it owns
	// (see DeclaredChannels). Everything else under it — role, curator,
	// reads/writes access lists — is still tolerated and ignored here; enforcing
	// the access contract is a separate, deferred concern (#167).
	// Values are kept raw because the shape varies across capabilities (an
	// object for `profile`, a bare string for `review`).
	Capabilities map[string]json.RawMessage `json:"capabilities"`
}

// DeclaredStores returns the durable-store paths this team declares under
// `capabilities.<name>.store`, in sorted capability order.
//
// A team that keeps state outside the reflected `.claude` tree — profile-team's
// `~/.atl/profiles` is the first — declares it here so the platform can version
// it without knowing which team it belongs to. Core stays team-agnostic: it
// honors a generic declaration, it does not learn a team name.
//
// Capability values whose shape carries no `store` (a bare string, an object
// without the field) are skipped rather than treated as an error — the same
// tolerance the rest of this struct applies to unknown fields.
func (t *TeamManifest) DeclaredStores() []string {
	var out []string
	for _, name := range sortedKeys(t.Capabilities) {
		var cap struct {
			Store string `json:"store"`
		}
		if err := json.Unmarshal(t.Capabilities[name], &cap); err != nil {
			continue // not an object, or otherwise unreadable — tolerate it
		}
		if s := strings.TrimSpace(cap.Store); s != "" {
			out = append(out, s)
		}
	}
	return out
}

// DeclaredChannels returns the capture channels this team declares under
// `capabilities.<name>.channel`, in sorted capability order — the exact sibling
// of DeclaredStores.
//
// A team that owns a capture channel — profile-team's `profile-fact` is the
// first — declares it here so the platform can emit that channel's per-turn
// signal without knowing which team owns it. The drain skill and the rule that
// acts on the signal both ship with the team; core honors a generic
// declaration, it does not learn a team name.
//
// Capability values whose shape carries no `channel` are skipped, as are
// declarations with a blank name (without a name there is nothing to record).
// A named declaration MISSING one of the other fields is returned anyway,
// deliberately: it is the only way the platform can later tell "declared but
// broken" from "not declared" without re-fetching the team source. Read-side
// callers refuse it loudly (see the channels doctor check).
func (t *TeamManifest) DeclaredChannels() []manifest.Channel {
	var out []manifest.Channel
	for _, name := range sortedKeys(t.Capabilities) {
		var cap struct {
			Channel *manifest.Channel `json:"channel"`
		}
		if err := json.Unmarshal(t.Capabilities[name], &cap); err != nil {
			continue // not an object, or otherwise unreadable — tolerate it
		}
		if cap.Channel == nil || strings.TrimSpace(cap.Channel.Name) == "" {
			continue
		}
		out = append(out, *cap.Channel)
	}
	return out
}

func sortedKeys(m map[string]json.RawMessage) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
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
