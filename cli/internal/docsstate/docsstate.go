// Package docsstate persists the docs-audit cursor at ~/.atl/docs-audit-state.json:
// the commit last fully audited and when. It exists solely for the /docs-audit
// backstop's pre-flight ("any doc-affecting commit since the last audit?") and its
// runaway-guard ("not again within ~1 day"). The deterministic `atl docs check` is
// stateless; coverage is computed live from the code tree, so there is no baseline
// to persist here — only the cursor.
package docsstate

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/agentteamland/atl/cli/internal/scope"
)

// SchemaVersion is the on-disk schema version of the state file.
const SchemaVersion = 1

// State is the docs-audit cursor.
type State struct {
	SchemaVersion int    `json:"schemaVersion"`
	LastAuditSHA  string `json:"lastAuditSHA"`
	LastAuditAt   string `json:"lastAuditAt"` // RFC3339, UTC
}

// Path returns ~/.atl/docs-audit-state.json.
func Path() (string, error) {
	dir, err := scope.LayerDir(scope.Global, "")
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "docs-audit-state.json"), nil
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

// Record stamps the given commit + time as the last audit and persists it.
func Record(sha string, t time.Time) error {
	s, err := Load()
	if err != nil {
		return err
	}
	s.LastAuditSHA = sha
	s.LastAuditAt = t.UTC().Format(time.RFC3339)
	return s.Save()
}
