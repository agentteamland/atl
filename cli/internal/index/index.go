// Package index resolves a team handle to its source — the read side of the
// GitHub-backed, zero-infra registry (decision doc item 4, "C executed as A").
//
// The index is a generated catalog: each entry maps a "<handle>/<name>" — where
// the handle is the GitHub owner, so repo ownership IS authorship — to a source
// (a standalone repo, or a subpath inside the atl monorepo for first-party
// teams), plus the publisher's default scope and an optional verified badge.
//
// Resolution is offline-first: a binary ships an embedded seed, a throttled
// network refresh keeps a ~/.atl/index.json cache fresh out of band, and Resolve
// prefers the cache but always falls back to the seed — so install never depends
// on the network being up.
package index

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

//go:embed index.json
var seedJSON []byte

// RawURL is where the published index lives. Until CI generates a dedicated
// artifact, it's the embedded seed's own repo path (public, so auth-free).
const RawURL = "https://raw.githubusercontent.com/agentteamland/atl/main/cli/internal/index/index.json"

var httpClient = &http.Client{Timeout: 30 * time.Second}

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

// CachePath returns ~/.atl/index.json — the network-refreshed index cache.
func CachePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".atl", "index.json"), nil
}

// Resolve returns the best index available WITHOUT touching the network: the
// refreshed cache if present and non-empty, otherwise the embedded seed. Install
// and update resolve handles through this; RefreshCache keeps the cache fresh
// out of band (throttled), so a stale or absent cache degrades to the seed
// rather than failing.
func Resolve() (*Index, error) {
	seed, serr := Seed()

	var cache *Index
	if path, err := CachePath(); err == nil {
		if b, rerr := os.ReadFile(path); rerr == nil {
			if ix, lerr := Load(b); lerr == nil && len(ix.Teams) > 0 {
				cache = ix
			}
		}
	}

	switch {
	case cache == nil:
		return seed, serr // no usable cache → the embedded seed
	case seed == nil || serr != nil:
		return cache, nil // seed unreadable (shouldn't happen) → the cache
	case newerGeneratedAt(seed.GeneratedAt, cache.GeneratedAt):
		// A freshly-upgraded binary can ship a NEWER catalog than a stale cache left
		// by an earlier `atl update`; prefer whichever was generated more recently so
		// the upgrade isn't masked by an out-of-date cache.
		return seed, nil
	default:
		return cache, nil
	}
}

// newerGeneratedAt reports whether generatedAt a is strictly newer than b. Both
// are RFC3339 stamps; if either is missing or unparsable the comparison is
// inconclusive and it returns false (keeping the historical cache-preference).
func newerGeneratedAt(a, b string) bool {
	ta, ea := time.Parse(time.RFC3339, a)
	tb, eb := time.Parse(time.RFC3339, b)
	if ea != nil || eb != nil {
		return false
	}
	return ta.After(tb)
}

// Fetch downloads and parses an index from url.
func Fetch(url string) (*Index, error) {
	resp, err := httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetch index: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch index: HTTP %d", resp.StatusCode)
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return Load(b)
}

// RefreshCache fetches the index from url and writes it to CachePath atomically.
// Best-effort: callers treat a failure as non-fatal (Resolve falls back).
func RefreshCache(url string) error {
	ix, err := Fetch(url)
	if err != nil {
		return err
	}
	path, err := CachePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(ix, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(b, '\n'), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
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

// LookupByName finds an entry by bare name (handle unknown), preferring a
// verified one when several handles publish the same name. Used to resolve a
// dependency declared as a bare team name rather than "<handle>/<name>".
func (ix *Index) LookupByName(name string) (*Entry, error) {
	var match *Entry
	for i := range ix.Teams {
		if ix.Teams[i].Name != name {
			continue
		}
		if ix.Teams[i].Verified {
			return &ix.Teams[i], nil // a verified match wins outright
		}
		if match == nil {
			match = &ix.Teams[i]
		}
	}
	if match == nil {
		return nil, fmt.Errorf("team %q not found in index", name)
	}
	return match, nil
}

// Search returns every entry whose handle, name, description, or one of its
// keywords contains query (case-insensitive). A blank query matches all teams,
// so the catalog can be browsed. Results keep the index's order.
func (ix *Index) Search(query string) []Entry {
	q := strings.ToLower(strings.TrimSpace(query))
	var out []Entry
	for _, e := range ix.Teams {
		if q == "" || entryMatchesQuery(e, q) {
			out = append(out, e)
		}
	}
	return out
}

func entryMatchesQuery(e Entry, q string) bool {
	if strings.Contains(strings.ToLower(e.Handle), q) ||
		strings.Contains(strings.ToLower(e.Name), q) ||
		strings.Contains(strings.ToLower(e.Description), q) {
		return true
	}
	for _, k := range e.Keywords {
		if strings.Contains(strings.ToLower(k), q) {
			return true
		}
	}
	return false
}

// ParseRef splits a "<handle>/<name>" reference.
func ParseRef(s string) (handle, name string, err error) {
	parts := strings.Split(s, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid team reference %q (want <handle>/<name>)", s)
	}
	return parts[0], parts[1], nil
}
