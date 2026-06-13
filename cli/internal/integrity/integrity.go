// Package integrity is the doctor's asset self-heal: it compares each install
// manifest against what is actually on disk and restores files that have gone
// missing — the deletion-recovery half of asset model (b) (#2b, 2026-06-14).
//
// An install manifest is a contract: these files MUST exist at this scope. A
// file the manifest lists but disk lacks is drift (an accidental delete, a fresh
// checkout) and is restored from the pinned source, checksum-verified. Only
// ABSENT files are restored — a file that is present but changed is a user edit
// (or a learning-loop evolution) and is preserved, never overwritten.
// Intentional removal goes through `atl remove`, which drops the manifest.
package integrity

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/agentteamland/atl/cli/internal/fanout"
	"github.com/agentteamland/atl/cli/internal/manifest"
	"github.com/agentteamland/atl/cli/internal/source"
)

// Missing is one manifest-listed file that is absent on disk.
type Missing struct {
	Handle string
	Name   string
	Rel    string // path relative to the .claude dir
	SHA    string // expected SHA-256 from the manifest
	Source manifest.Source
}

// Scan returns every manifest-listed file under layerDir that is absent from
// claudeDir. Present-but-changed files are NOT reported — they are user edits to
// preserve, not drift to heal.
func Scan(layerDir, claudeDir string) ([]Missing, error) {
	manifests, err := manifest.List(layerDir)
	if err != nil {
		return nil, err
	}
	var missing []Missing
	for _, m := range manifests {
		for rel, sha := range m.Files {
			h, err := fanout.HashFile(filepath.Join(claudeDir, rel))
			if err != nil {
				return nil, err
			}
			if h == "" { // absent on disk
				missing = append(missing, Missing{
					Handle: m.Handle, Name: m.Name, Rel: rel, SHA: sha, Source: m.Source,
				})
			}
		}
	}
	return missing, nil
}

// RestoreAll re-fetches the missing files (one fetch per distinct source) and
// writes them back into claudeDir, checksum-verified. Returns how many were
// restored.
func RestoreAll(missing []Missing, claudeDir string) (int, error) {
	type key struct{ repo, subpath, ref string }
	groups := map[key][]Missing{}
	for _, m := range missing {
		groups[key{m.Source.Repo, m.Source.Subpath, m.Source.Ref}] = append(
			groups[key{m.Source.Repo, m.Source.Subpath, m.Source.Ref}], m)
	}
	restored := 0
	for k, ms := range groups {
		srcDir, err := source.Fetch(k.repo, k.subpath, k.ref)
		if err != nil {
			return restored, err
		}
		for _, m := range ms {
			if err := restoreFromDir(srcDir, m, claudeDir); err != nil {
				os.RemoveAll(srcDir)
				return restored, err
			}
			restored++
		}
		os.RemoveAll(srcDir)
	}
	return restored, nil
}

// restoreFromDir copies one missing file from a fetched source dir into
// claudeDir, verifying it matches the manifest's recorded hash.
func restoreFromDir(srcDir string, m Missing, claudeDir string) error {
	b, err := os.ReadFile(filepath.Join(srcDir, m.Rel))
	if err != nil {
		return fmt.Errorf("restore %s: %w", m.Rel, err)
	}
	if fanout.Hash(b) != m.SHA {
		return fmt.Errorf("restore %s: checksum mismatch (source changed?)", m.Rel)
	}
	dst := filepath.Join(claudeDir, m.Rel)
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	return os.WriteFile(dst, b, 0o644)
}
