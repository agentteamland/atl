// Package manifest records what `atl install` wrote, so the rest of the
// platform can reason about an install deterministically.
//
// One JSON file per installed team, at
// <layer>/.atl/installed/<handle>__<name>.json, where <layer> is the scope root
// (~/.atl for global, <project>/.atl for project). It is a dual-role contract:
//
//   - fanout baseline: the files map holds the SHA-256 each file had at install
//     time, so update's three-way refresh tells "unmodified" from "the user
//     changed it" (internal/fanout).
//   - integrity contract: the set of files that MUST exist at this scope, so
//     doctor can detect a deleted/corrupted copy and restore it.
package manifest

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// SchemaVersion is the current install-manifest schema.
const SchemaVersion = 1

// Source mirrors the index source an install resolved from, pinned to the ref
// actually fetched, so doctor/update can re-fetch the exact same bytes.
type Source struct {
	Repo    string `json:"repo"`
	Subpath string `json:"subpath"`
	Ref     string `json:"ref"`
}

// Manifest is the record of one installed team at one scope.
type Manifest struct {
	SchemaVersion int               `json:"schemaVersion"`
	Handle        string            `json:"handle"`
	Name          string            `json:"name"`
	Version       string            `json:"version"`
	Scope         string            `json:"scope"` // the effective scope installed at
	Source        Source            `json:"source"`
	InstalledAt   time.Time         `json:"installedAt"`
	Files         map[string]string `json:"files"` // path relative to the .claude dir -> SHA-256 at install
}

// dirName is the installed-manifests directory under a layer root.
const dirName = "installed"

func fileName(handle, name string) string { return handle + "__" + name + ".json" }

// Path returns the manifest path for a team under layerDir (a scope root such
// as ~/.atl or <project>/.atl).
func Path(layerDir, handle, name string) string {
	return filepath.Join(layerDir, dirName, fileName(handle, name))
}

// Write atomically writes m under layerDir. SchemaVersion and InstalledAt are
// filled if unset.
func (m *Manifest) Write(layerDir string) error {
	if m.SchemaVersion == 0 {
		m.SchemaVersion = SchemaVersion
	}
	if m.InstalledAt.IsZero() {
		m.InstalledAt = time.Now().UTC()
	}
	path := Path(layerDir, m.Handle, m.Name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(b, '\n'), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// Read loads a team's manifest from layerDir.
func Read(layerDir, handle, name string) (*Manifest, error) {
	b, err := os.ReadFile(Path(layerDir, handle, name))
	if err != nil {
		return nil, err
	}
	var m Manifest
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, fmt.Errorf("parse manifest %s/%s: %w", handle, name, err)
	}
	return &m, nil
}

// List returns every installed-team manifest under layerDir, sorted by ref
// (empty if the layer has no installs yet).
func List(layerDir string) ([]*Manifest, error) {
	dir := filepath.Join(layerDir, dirName)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []*Manifest
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		var m Manifest
		if err := json.Unmarshal(b, &m); err != nil {
			return nil, fmt.Errorf("parse manifest %s: %w", e.Name(), err)
		}
		out = append(out, &m)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Handle+"/"+out[i].Name < out[j].Handle+"/"+out[j].Name
	})
	return out, nil
}

// Remove deletes a team's manifest from layerDir. Idempotent.
func Remove(layerDir, handle, name string) error {
	err := os.Remove(Path(layerDir, handle, name))
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
