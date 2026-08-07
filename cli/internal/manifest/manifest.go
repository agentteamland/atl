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
	"strings"
	"time"
)

// SchemaVersion is the current install-manifest schema.
//
//	1 — the original: source pin + files baseline.
//	2 — adds `stores`, the durable-store paths the team declared. An install
//	    written at v1 predates the field, so its stores are unknown rather than
//	    absent — update backfills those once and stamps v2.
//	3 — adds `channels`, the capture channels the team declared. Like `stores`,
//	    the value lives only in the team's team.json, so an install written at
//	    v2 must re-fetch its pinned source once to learn it.
//	4 — adds `reviewer`, the agent the team offers /create-pr for domain review.
//	5 — adds `sessionScripts`, the scripts the team declared for session start.
//	    Same story as its three siblings: the declaration lives only in team.json,
//	    which install does not reflect onto disk, so an install written at v4 has
//	    to re-fetch its pinned source once to learn it.
const SchemaVersion = 5

// Source mirrors the index source an install resolved from, pinned to the ref
// actually fetched, so doctor/update can re-fetch the exact same bytes.
type Source struct {
	Repo    string `json:"repo"`
	Subpath string `json:"subpath"`
	Ref     string `json:"ref"`
}

// Channel is one capture channel a team declares under
// `capabilities.<name>.channel`. Core emits the signal for it and refuses
// markers on any channel no installed team declares; the drain skill and the
// rule that acts on the signal both ship with the owning team, so core names
// no team.
type Channel struct {
	Name      string `json:"name"`      // queue channel + marker prefix, e.g. "profile-fact"
	Drain     string `json:"drain"`     // the skill a background subagent runs, e.g. "/profile-drain"
	Rule      string `json:"rule"`      // the rule that carries the action instruction
	Describes string `json:"describes"` // human label for the watchdog's "missed <…>"
}

// Valid reports whether every field a signal sentence needs is present. An
// invalid declaration is refused rather than emitted with a hole in it.
func (c Channel) Valid() bool { return len(c.MissingFields()) == 0 }

// MissingFields lists the fields a signal sentence needs and this declaration
// does not carry, in declaration order — the detail a doctor warning reports so
// a broken declaration says what is wrong with it, not just that it was refused.
func (c Channel) MissingFields() []string {
	var out []string
	for _, f := range []struct {
		name  string
		value string
	}{
		{"name", c.Name}, {"drain", c.Drain}, {"rule", c.Rule}, {"describes", c.Describes},
	} {
		if strings.TrimSpace(f.value) == "" {
			out = append(out, f.name)
		}
	}
	return out
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
	// Stores are the durable-store paths this team declared under
	// `capabilities.<name>.store` in its team.json, verbatim (still tilde-form).
	// Recorded so the platform can keep them versioned without knowing which
	// team owns them — core honors the declaration, it does not learn the team.
	// Omitted for teams that declare none.
	Stores []string `json:"stores,omitempty"`
	// Channels are the capture channels this team declared under
	// `capabilities.<name>.channel`. Recorded so the platform can emit each
	// channel's signal without knowing which team owns it. Omitted for teams that
	// declare none.
	Channels []Channel `json:"channels,omitempty"`
	// Reviewer is the agent this team declared under `capabilities.review` — the
	// domain specialist /create-pr spawns over a diff alongside its generic
	// reviewer. Recorded here for the same reason as Stores and Channels: the
	// declaration lives in team.json, which is NOT an installable asset, so a
	// consumer that only sees the installed tree cannot read it from disk.
	// Empty for teams that declare none, which is the common case.
	Reviewer string `json:"reviewer,omitempty"`
	// SessionScripts are the scripts this team declared under
	// `capabilities.<name>.sessionScript` — each a path relative to the team's
	// asset root (so `scripts/x.sh` resolves to <scope>/.claude/scripts/x.sh once
	// reflected). Session start runs each declared script and forwards its stdout
	// into the session's context.
	//
	// Recorded here for the same reason as Stores, Channels and Reviewer: the
	// declaration lives in team.json, which is NOT an installable asset, so a
	// consumer that only sees the installed tree cannot read it from disk.
	// Omitted for teams that declare none.
	SessionScripts []string `json:"sessionScripts,omitempty"`
}

// dirName is the installed-manifests directory under a layer root.
const dirName = "installed"

func fileName(handle, name string) string { return handle + "__" + name + ".json" }

// Path returns the manifest path for a team under layerDir (a scope root such
// as ~/.atl or <project>/.atl).
func Path(layerDir, handle, name string) string {
	return filepath.Join(layerDir, dirName, fileName(handle, name))
}

// Write atomically writes m under layerDir. InstalledAt is filled if unset, and
// SchemaVersion only when it is zero.
//
// Write deliberately does NOT stamp the current schema onto an existing
// manifest. Several paths read a manifest, mutate one field, and write it back —
// fan-out and promote both rewrite only Files — and if those advanced the schema
// they would claim a shape they never produced, permanently skipping the
// one-time migration meant to fill the rest. Claiming the current schema belongs
// to the paths that build the whole record: install and update.
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
