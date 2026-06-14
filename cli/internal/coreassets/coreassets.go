// Package coreassets ships the platform's core rules + skills inside the atl
// binary and reflects them into the user-global Claude Code directory.
//
// Core is the platform layer: unlike teams (resolved from the index and fetched
// per-install), core is the same for every user and rides inside the binary —
// built from the same monorepo, so the embedded copy is always in lockstep with
// the binary version. go:embed cannot reach outside the cli/ module, so core/ is
// mirrored into embed/ by sync.sh before build (see TestEmbedMatchesCore).
package coreassets

import (
	"embed"
	"io/fs"
	"os"
	"path/filepath"
)

//go:embed all:embed
var assets embed.FS

const embedRoot = "embed"

// Reflect writes the embedded core rules + skills into the global Claude dir
// (e.g. ~/.claude/rules, ~/.claude/skills). Refresh-always: core is the
// platform layer and is not user-owned — a user who wants to override a rule
// does it at project scope, which shadows global. Returns the file count.
func Reflect(globalClaudeDir string) (int, error) {
	written := 0
	err := fs.WalkDir(assets, embedRoot, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(embedRoot, p) // e.g. rules/communication-style.md
		if err != nil {
			return err
		}
		b, err := assets.ReadFile(p)
		if err != nil {
			return err
		}
		dst := filepath.Join(globalClaudeDir, filepath.FromSlash(rel))
		if existing, rerr := os.ReadFile(dst); rerr == nil && string(existing) == string(b) {
			return nil // unchanged — skip the write so callers see real change counts
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(dst, b, 0o644); err != nil {
			return err
		}
		written++
		return nil
	})
	return written, err
}
