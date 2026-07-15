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

//go:embed templates/global.md templates/project.md templates/monorepo.md templates/backlog.md templates/tasks.md
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
	body, err := Skeleton(tier, name)
	if err != nil {
		return path, false, err
	}
	created, err = writeIfAbsent(path, body)
	return path, created, err
}

// stateFiles are the per-project decision-state files scaffolded under .atl/:
// backlog.md (the deferred, trigger-gated superset) + tasks.md (the active-intent
// subset). See the brainstorm rule's "Backlog + tasks discipline".
var stateFiles = []string{"backlog.md", "tasks.md"}

// WriteStateFilesIfAbsent drops the .atl/backlog.md + .atl/tasks.md skeletons under
// root, each only if absent (a user's own file is never overwritten). It returns the
// paths it actually created. These are project-scoped decision state — callers must
// NOT invoke this for the global tier (which has no project .atl/).
func WriteStateFilesIfAbsent(root string) (created []string, err error) {
	atlDir := filepath.Join(root, ".atl")
	for _, name := range stateFiles {
		b, rerr := templates.ReadFile("templates/" + name)
		if rerr != nil {
			return created, rerr
		}
		path := filepath.Join(atlDir, name)
		did, werr := writeIfAbsent(path, string(b))
		if werr != nil {
			return created, werr
		}
		if did {
			created = append(created, path)
		}
	}
	return created, nil
}

// writeIfAbsent writes body to path only if nothing exists there yet, creating
// parent directories as needed. It returns whether it wrote (false = a file already
// existed and was left untouched — never clobber user-owned content).
func writeIfAbsent(path, body string) (created bool, err error) {
	if _, serr := os.Stat(path); serr == nil {
		return false, nil
	} else if !os.IsNotExist(serr) {
		return false, serr
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return false, err
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		return false, err
	}
	return true, nil
}
