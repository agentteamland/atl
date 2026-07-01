// Package rulesstate persists the rules-distill cursor at
// ~/.atl/rules-distill-state.json: the commit last distilled by /rules-distill
// and when. It mirrors skillsstate/docsstate — it exists for the /rules-distill
// pre-flight ("any corpus-affecting commit since the last distill?") and its
// ~1-day runaway-guard. The deterministic collect (`atl rules scan`) is stateless;
// only the LLM distill needs a cursor.
package rulesstate

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/agentteamland/atl/cli/internal/scope"
)

// SchemaVersion is the on-disk schema version of the state file.
const SchemaVersion = 1

// State is the rules-distill cursor.
type State struct {
	SchemaVersion  int    `json:"schemaVersion"`
	LastDistillSHA string `json:"lastDistillSHA"`
	LastDistillAt  string `json:"lastDistillAt"` // RFC3339, UTC
}

// Path returns ~/.atl/rules-distill-state.json.
func Path() (string, error) {
	dir, err := scope.LayerDir(scope.Global, "")
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "rules-distill-state.json"), nil
}

// Load reads the state. A missing file is an empty state, not an error.
func Load() (*State, error) {
	p, err := Path()
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return &State{SchemaVersion: SchemaVersion}, nil
		}
		return nil, err
	}
	var s State
	if err := json.Unmarshal(b, &s); err != nil {
		return nil, err
	}
	if s.SchemaVersion == 0 {
		s.SchemaVersion = SchemaVersion
	}
	return &s, nil
}

// Save atomically persists the state (tmp + rename).
func (s *State) Save() error {
	if s.SchemaVersion == 0 {
		s.SchemaVersion = SchemaVersion
	}
	p, err := Path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, append(b, '\n'), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}

// Record stamps the given commit + time as the last distill and persists it.
func Record(sha string, t time.Time) error {
	s, err := Load()
	if err != nil {
		return err
	}
	s.LastDistillSHA = sha
	s.LastDistillAt = t.UTC().Format(time.RFC3339)
	return s.Save()
}
