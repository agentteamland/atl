// Package userrules reflects a scope's user-authored rules — the ones /rule
// writes to <layer>/.atl/rules — into that scope's Claude Code load surface
// (<layer>/.claude/rules), which is where Claude Code actually loads rules at
// session start. Without this reflection a rule authored via `/rule` or
// `/rule --global` lands in .atl/rules and is never loaded: this package is the
// missing load path.
//
// It mirrors coreassets.Reflect (durable source → reflected consumption): the
// .atl/rules file is the durable source of truth, the .claude/rules copy is
// derived. gc treats a reflected copy as owned exactly while its .atl/rules
// source still exists (see gc.scanLayer via Owned), so deleting the source is
// what lets gc reclaim the stale copy — the reclamation stays coherent.
package userrules

import (
	"os"
	"path/filepath"
	"strings"
)

// Reflect copies every *.md rule from <layerDir>/rules into <claudeDir>/rules.
// It is additive and idempotent: an unchanged destination is skipped (so callers
// see real change counts) and no file outside <claudeDir>/rules is touched. A
// name in `protected` is skipped — used at the global layer to stop a user rule
// from clobbering a same-named core rule (core is authoritative at global; a
// user overrides at project scope instead). A missing source dir is not an
// error. Returns the count of files actually written.
func Reflect(layerDir, claudeDir string, protected map[string]bool) (int, error) {
	srcDir := filepath.Join(layerDir, "rules")
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil // no user rules at this layer — nothing to reflect
		}
		return 0, err
	}
	dstDir := filepath.Join(claudeDir, "rules")
	written := 0
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".md") {
			continue
		}
		if protected[name] {
			continue
		}
		b, rerr := os.ReadFile(filepath.Join(srcDir, name))
		if rerr != nil {
			return written, rerr
		}
		dst := filepath.Join(dstDir, name)
		if existing, eerr := os.ReadFile(dst); eerr == nil && string(existing) == string(b) {
			continue // unchanged — skip so callers see real change counts
		}
		if err := os.MkdirAll(dstDir, 0o755); err != nil {
			return written, err
		}
		if err := os.WriteFile(dst, b, 0o644); err != nil {
			return written, err
		}
		written++
	}
	return written, nil
}

// Owned returns the rule paths — relative to the Claude dir, e.g.
// "rules/house-style.md" — that the user has authored at this layer: the .md
// files under <layerDir>/rules. gc uses this to treat their reflected copies as
// owned so it never reclaims a user's own global or project rules. A missing dir
// yields an empty set, not an error.
func Owned(layerDir string) (map[string]bool, error) {
	srcDir := filepath.Join(layerDir, "rules")
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]bool{}, nil
		}
		return nil, err
	}
	out := map[string]bool{}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".md") {
			continue
		}
		out["rules/"+name] = true
	}
	return out, nil
}
