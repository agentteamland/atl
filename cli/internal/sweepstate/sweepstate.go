// Package sweepstate persists the per-sweep cursor for ATL's LLM-backstop sweeps —
// /docs-audit, /skill-stocktake, /rules-distill, /observe — each recording the commit
// last swept and when, under ~/.atl/<kind>-state.json. The deterministic CLI halves
// (`atl docs check`, `atl skills check`, `atl rules scan`, `atl observe`) are stateless;
// only the LLM sweeps need a cursor. It also answers "is a full sweep due?": a ~1-day
// runaway-guard over any commit touching the kind's scanned paths since the last
// recorded sweep. One generic cursor replaces the former docsstate/skillsstate/
// rulesstate triple.
package sweepstate

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/agentteamland/atl/cli/internal/scope"
)

// SchemaVersion is the on-disk schema version of every cursor file.
const SchemaVersion = 1

// Cursor is one sweep's persisted state: the commit last swept and when.
type Cursor struct {
	SchemaVersion int    `json:"schemaVersion"`
	LastSHA       string `json:"lastSHA"`
	LastAt        string `json:"lastAt"` // RFC3339, UTC
}

// Kind is a sweep's fixed identity: the file it persists to and the repo paths a
// commit must touch to count as "affecting" it. Callers use the package vars below;
// new kinds are declared here, never constructed ad hoc.
type Kind struct {
	filename  string
	scanPaths []string
}

var (
	// Docs is the /docs-audit cursor (~/.atl/docs-audit-state.json).
	Docs = Kind{filename: "docs-audit-state.json", scanPaths: []string{"docs", "core", "cli"}}
	// Skills is the /skill-stocktake cursor (~/.atl/skill-stocktake-state.json).
	Skills = Kind{filename: "skill-stocktake-state.json", scanPaths: []string{"core", "teams"}}
	// Rules is the /rules-distill cursor (~/.atl/rules-distill-state.json).
	Rules = Kind{filename: "rules-distill-state.json", scanPaths: []string{"core", "teams"}}
	// Observe is the /observe cursor (~/.atl/observe-state.json). Its scope is the
	// project's ATL decision + knowledge surface (.atl/ — brainstorms, docs, journal),
	// which moves whenever work ships or a decision lands, so a proactive-observer sweep
	// becomes due right when shipped-vs-designed drift is most likely.
	Observe = Kind{filename: "observe-state.json", scanPaths: []string{".atl"}}
)

// Path returns ~/.atl/<kind>-state.json.
func (k Kind) Path() (string, error) {
	dir, err := scope.LayerDir(scope.Global, "")
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, k.filename), nil
}

// Load reads the cursor. A missing file is an empty cursor, not an error.
func (k Kind) Load() (*Cursor, error) {
	p, err := k.Path()
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return &Cursor{SchemaVersion: SchemaVersion}, nil
		}
		return nil, err
	}
	var c Cursor
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, err
	}
	if c.SchemaVersion == 0 {
		c.SchemaVersion = SchemaVersion
	}
	return &c, nil
}

// Save atomically persists the cursor (tmp + rename).
func (k Kind) Save(c *Cursor) error {
	if c.SchemaVersion == 0 {
		c.SchemaVersion = SchemaVersion
	}
	p, err := k.Path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, append(b, '\n'), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}

// Record stamps the given commit + time as the last sweep and persists it.
func (k Kind) Record(sha string, t time.Time) error {
	c, err := k.Load()
	if err != nil {
		return err
	}
	c.LastSHA = sha
	c.LastAt = t.UTC().Format(time.RFC3339)
	return k.Save(c)
}

// Due reports whether a full sweep is due for this kind: a commit touching one of the
// kind's scanned paths has landed since the recorded cursor, gated by a ~1-day
// runaway-guard. Best-effort — any error (unreadable cursor, non-git dir) yields
// false so a session-start signal never blocks or spams.
func (k Kind) Due(repoRoot string) bool {
	c, err := k.Load()
	if err != nil {
		return false
	}
	if c.LastAt != "" {
		if t, perr := time.Parse(time.RFC3339, c.LastAt); perr == nil && time.Since(t) < 24*time.Hour {
			return false // runaway-guard: don't sweep again within ~1 day
		}
	}
	return k.affectingCommitsSince(repoRoot, c.LastSHA)
}

// affectingCommitsSince reports whether any commit touching one of the kind's scanned
// paths has landed since sinceSHA (or, when sinceSHA is empty, whether the repo has
// any such commit at all — i.e. it was never swept).
func (k Kind) affectingCommitsSince(repoRoot, sinceSHA string) bool {
	run := func(sha string) ([]byte, error) {
		args := []string{"-C", repoRoot, "log", "--oneline", "-1"}
		if sha != "" {
			args = append(args, sha+"..HEAD")
		}
		args = append(args, "--")
		args = append(args, k.scanPaths...)
		return exec.Command("git", args...).Output()
	}
	out, err := run(sinceSHA)
	if err != nil {
		// A recorded SHA git no longer knows (rebase / squash / gc pruned it) makes
		// `<sha>..HEAD` fail — treat that as never-swept (retry with no range) so the
		// backstop doesn't go permanently silent, rather than reading the git failure
		// as "nothing changed". A failure with no range means it's not a git repo (or
		// has no history), where there is genuinely nothing to sweep.
		if sinceSHA == "" {
			return false
		}
		if out, err = run(""); err != nil {
			return false
		}
	}
	return len(strings.TrimSpace(string(out))) > 0
}
