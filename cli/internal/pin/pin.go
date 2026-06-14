// Package pin records project-local promotion opt-outs — paths the user wants
// kept project-only, never lifted to the global layer by `atl promote`.
//
// One JSON file per project, at <project>/.atl/pins.json. A pin is a path
// relative to the project's .claude dir (slash-separated), naming either a file
// or a subtree (an agent/skill/rule unit). promote skips any file equal to a pin
// or nested under one. Fan-out is unaffected — a pinned divergence is already
// preserved on the pull side; pin only stops the upward lift. A pin is a
// declarative exclusion (like .gitignore), not a manual step: promotion still
// runs automatically; pin scopes it.
package pin

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// SchemaVersion is the current pins-file schema.
const SchemaVersion = 1

const fileName = "pins.json"

// Set is one project's pin list.
type Set struct {
	SchemaVersion int      `json:"schemaVersion"`
	Pins          []string `json:"pins"`
}

// Path returns the pins file under layerDir (a project's .atl root).
func Path(layerDir string) string { return filepath.Join(layerDir, fileName) }

// Load reads the pin set from layerDir. A missing file is an empty set, not an
// error — most projects pin nothing.
func Load(layerDir string) (*Set, error) {
	b, err := os.ReadFile(Path(layerDir))
	if err != nil {
		if os.IsNotExist(err) {
			return &Set{SchemaVersion: SchemaVersion}, nil
		}
		return nil, err
	}
	var s Set
	if err := json.Unmarshal(b, &s); err != nil {
		return nil, err
	}
	if s.SchemaVersion == 0 {
		s.SchemaVersion = SchemaVersion
	}
	return &s, nil
}

// Write atomically persists the set under layerDir, pins sorted for a stable
// on-disk order.
func (s *Set) Write(layerDir string) error {
	if s.SchemaVersion == 0 {
		s.SchemaVersion = SchemaVersion
	}
	sort.Strings(s.Pins)
	if err := os.MkdirAll(layerDir, 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	path := Path(layerDir)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(b, '\n'), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// Normalize cleans a user-supplied path into the canonical .claude-relative,
// slash form used as a pin key.
func Normalize(rel string) string {
	rel = filepath.ToSlash(filepath.Clean(strings.TrimSpace(rel)))
	rel = strings.TrimPrefix(rel, "./")
	rel = strings.Trim(rel, "/")
	return rel
}

// Add inserts a pin, returning false if it was already present.
func (s *Set) Add(rel string) bool {
	rel = Normalize(rel)
	for _, p := range s.Pins {
		if p == rel {
			return false
		}
	}
	s.Pins = append(s.Pins, rel)
	return true
}

// Remove drops a pin, returning false if it wasn't present.
func (s *Set) Remove(rel string) bool {
	rel = Normalize(rel)
	for i, p := range s.Pins {
		if p == rel {
			s.Pins = append(s.Pins[:i], s.Pins[i+1:]...)
			return true
		}
	}
	return false
}

// Pinned reports whether a .claude-relative file path is exempt from promotion —
// it equals a pin or is nested under a pinned subtree.
func (s *Set) Pinned(rel string) bool {
	rel = Normalize(rel)
	for _, p := range s.Pins {
		if rel == p || strings.HasPrefix(rel, p+"/") {
			return true
		}
	}
	return false
}
