// Package index resolves a team handle to its source — the read side of the
// GitHub-backed, zero-infra registry (decision doc item 4, "C executed as A").
//
// The index is a generated catalog: each entry maps a "<handle>/<name>" — where
// the handle is the GitHub owner, so repo ownership IS authorship — to a source
// (a standalone repo, or a subpath inside the atl monorepo for first-party
// teams), plus the publisher's default scope and an optional verified badge.
// v1 ships an embedded seed; a later step fetches a fresher index over the
// network without changing this resolve surface.
package index

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
)

//go:embed index.json
var seedJSON []byte

// Source locates a team's files. Subpath is "" for a standalone repo (the
// third-party idiom); for a first-party team it points inside the atl monorepo.
type Source struct {
	Repo    string `json:"repo"`    // "<owner>/<repo>"
	Subpath string `json:"subpath"` // "" = repo root
	Ref     string `json:"ref"`     // tag, branch, or commit SHA
}

// Entry is one catalogued team.
type Entry struct {
	Handle      string   `json:"handle"` // GitHub owner = authorship proof
	Name        string   `json:"name"`   // unique within the handle
	Version     string   `json:"version"`
	Description string   `json:"description,omitempty"`
	Keywords    []string `json:"keywords,omitempty"`
	Scope       string   `json:"scope,omitempty"` // "" | project | global | both
	Verified    bool     `json:"verified,omitempty"`
	Source      Source   `json:"source"`
}

// Ref is the "<handle>/<name>" install reference.
func (e Entry) Ref() string { return e.Handle + "/" + e.Name }

// Index is the generated catalog.
type Index struct {
	SchemaVersion int     `json:"schemaVersion"`
	GeneratedAt   string  `json:"generatedAt,omitempty"`
	Teams         []Entry `json:"teams"`
}

// Load parses an index from JSON bytes.
func Load(b []byte) (*Index, error) {
	var ix Index
	if err := json.Unmarshal(b, &ix); err != nil {
		return nil, fmt.Errorf("parse index: %w", err)
	}
	return &ix, nil
}

// Seed returns the embedded index shipped with this binary.
func Seed() (*Index, error) {
	return Load(seedJSON)
}

// Lookup finds the entry for "<handle>/<name>".
func (ix *Index) Lookup(handle, name string) (*Entry, error) {
	for i := range ix.Teams {
		if ix.Teams[i].Handle == handle && ix.Teams[i].Name == name {
			return &ix.Teams[i], nil
		}
	}
	return nil, fmt.Errorf("team %q not found in index", handle+"/"+name)
}

// ParseRef splits a "<handle>/<name>" reference.
func ParseRef(s string) (handle, name string, err error) {
	parts := strings.Split(s, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid team reference %q (want <handle>/<name>)", s)
	}
	return parts[0], parts[1], nil
}
