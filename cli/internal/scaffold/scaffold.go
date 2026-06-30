// Package scaffold drops starter CLAUDE.md files — one lean skeleton per tier
// (global persona / project hybrid / monorepo lean), per the CLAUDE.md template
// decision. It only ever writes when no CLAUDE.md exists at the target, so a
// user's own file is never overwritten; ongoing managed-block maintenance stays
// with the /brainstorm and /drain skills (no deterministic injector here).
package scaffold

import (
	"embed"
	"os"
	"path/filepath"
	"strings"
)

//go:embed templates/global.md templates/project.md templates/monorepo.md
var templates embed.FS

// Tier is which CLAUDE.md skeleton to drop.
type Tier string

const (
	Global   Tier = "global"   // ~/.claude/CLAUDE.md — pure user persona
	Project  Tier = "project"  // <root>/CLAUDE.md — hybrid (managed blocks + owned facts)
	Monorepo Tier = "monorepo" // <root>/CLAUDE.md — lean orientation (pointers only)
)

// Skeleton returns the starter CLAUDE.md content for a tier. If name is non-empty
// it fills the {{NAME}} placeholder (the project / repo name).
func Skeleton(tier Tier, name string) (string, error) {
	b, err := templates.ReadFile("templates/" + string(tier) + ".md")
	if err != nil {
		return "", err
	}
	s := string(b)
	if name != "" {
		s = strings.ReplaceAll(s, "{{NAME}}", name)
	}
	return s, nil
}

// Path returns the CLAUDE.md location for a tier. Global lives inside ~/.claude;
// project and monorepo live at the project ROOT (one level above <root>/.claude).
func Path(tier Tier, root string) (string, error) {
	if tier == Global {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, ".claude", "CLAUDE.md"), nil
	}
	return filepath.Join(root, "CLAUDE.md"), nil
}

// WriteIfAbsent writes the tier skeleton only if no CLAUDE.md exists at its path.
// It returns the path, whether it created the file (false = one already existed and
// was left untouched, since it is user-owned), and any error.
func WriteIfAbsent(tier Tier, root, name string) (path string, created bool, err error) {
	path, err = Path(tier, root)
	if err != nil {
		return "", false, err
	}
	if _, serr := os.Stat(path); serr == nil {
		return path, false, nil // exists — never clobber user-owned content
	} else if !os.IsNotExist(serr) {
		return path, false, serr
	}
	body, err := Skeleton(tier, name)
	if err != nil {
		return path, false, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return path, false, err
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		return path, false, err
	}
	return path, true, nil
}
