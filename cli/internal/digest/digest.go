// Package digest is where a sweep leaves a finding that needs a human decision.
//
// # Why this exists as a store rather than a message
//
// The sweep-dispatch decision splits a sweep's output by whether a finding needs
// a decision: actionable-and-already-decided goes to the board, which already
// carries work to closure; needing-judgement goes to the user. Two shapes for
// that second half were rejected, and this package is what is left after both.
//
//   - Speaking straight into the chat interrupts on the SWEEP's schedule rather
//     than the reader's. That is the constant-channel shape this project has
//     measured: a mechanism that speaks every time it runs teaches the reader to
//     skip it.
//   - A file alone is a durable artifact whose reading is unenforced — state and
//     a check with nothing dispatching, which reproduces the very failure the
//     sweep-dispatch work was about.
//
// So: a durable store PLUS an unread count in the session signal. The count is
// deliberately all the signal carries. A signal that restates its payload every
// session is the first rejected shape wearing the second's clothes.
//
// # Idempotence
//
// A sweep re-runs whenever its paths move, and a latent gap does not stop being
// a latent gap because it was reported yesterday. Findings are therefore keyed
// by a stable hash of (sweep, title): re-reporting one converges on the existing
// entry instead of growing the digest, which is what keeps a daily sweep from
// re-teaching the reader to skip it.
package digest

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/agentteamland/atl/cli/internal/scope"
)

// SchemaVersion is the on-disk schema version of a digest file.
const SchemaVersion = 1

// Finding is one thing a sweep wants a human to decide.
type Finding struct {
	// ID is derived from (Sweep, Title) — see NewID. Stable across runs so a
	// re-reporting sweep updates rather than duplicates.
	ID    string `json:"id"`
	Sweep string `json:"sweep"` // "observe", "skill-stocktake", ...
	Title string `json:"title"`
	Body  string `json:"body"`

	AddedAt time.Time `json:"addedAt"`
	// ReadAt is nil while the finding is unread. Reading does not delete: a
	// finding the reader has seen but not yet decided on is still real, and a
	// digest that empties itself on being read would lose exactly the ones that
	// needed thinking about.
	ReadAt *time.Time `json:"readAt,omitempty"`
}

// Unread reports whether this finding still awaits a first read.
func (f Finding) Unread() bool { return f.ReadAt == nil }

// NewID derives a finding's stable key. Deliberately (sweep, title) and NOT the
// body: a sweep that re-words its evidence between runs is reporting the same
// finding, and keying on the body would make every rewording a new entry.
func NewID(sweep, title string) string {
	sum := sha1.Sum([]byte(sweep + "\x00" + title))
	return hex.EncodeToString(sum[:])[:16]
}

type file struct {
	SchemaVersion int       `json:"schemaVersion"`
	Findings      []Finding `json:"findings"`
}

// Path returns the digest file for a project, under ~/.atl/digest/<hash>.json.
//
// Per project, for the same reason /observe's cursor is: a sweep fires in every
// project with an .atl/ surface, so one global file would let whichever project
// was opened first answer for all the others.
func Path(projectRoot string) (string, error) {
	dir, err := scope.LayerDir(scope.Global, "")
	if err != nil {
		return "", err
	}
	sum := sha1.Sum([]byte(filepath.Clean(projectRoot)))
	return filepath.Join(dir, "digest", hex.EncodeToString(sum[:])[:16]+".json"), nil
}

// Load reads a project's findings, newest first. A missing file is an empty
// digest, not an error — every project starts here.
func Load(projectRoot string) ([]Finding, error) {
	p, err := Path(projectRoot)
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var f file
	if err := json.Unmarshal(b, &f); err != nil {
		// A corrupt digest must not wedge the sweep that writes it, nor the
		// session signal that reads it. Treat it as empty and let the next Add
		// rewrite it: losing a finding is recoverable (the sweep re-reports it),
		// a permanently failing read is not.
		return nil, nil
	}
	sort.SliceStable(f.Findings, func(i, j int) bool {
		return f.Findings[i].AddedAt.After(f.Findings[j].AddedAt)
	})
	return f.Findings, nil
}

// Add records a finding, returning false when one with the same ID is already
// present. An existing finding is refreshed in place — its body is updated to
// the latest wording — but its ReadAt is PRESERVED: re-reporting must not make a
// finding the reader has already seen unread again, which would turn every sweep
// into a fresh interruption about the same thing.
func Add(projectRoot string, f Finding) (added bool, err error) {
	if f.ID == "" {
		f.ID = NewID(f.Sweep, f.Title)
	}
	if f.AddedAt.IsZero() {
		f.AddedAt = time.Now().UTC()
	}
	findings, err := Load(projectRoot)
	if err != nil {
		return false, err
	}
	for i := range findings {
		if findings[i].ID == f.ID {
			findings[i].Body = f.Body
			return false, save(projectRoot, findings)
		}
	}
	return true, save(projectRoot, append(findings, f))
}

// MarkRead stamps every currently-unread finding as read and reports how many
// changed. Called after the findings have actually been shown.
func MarkRead(projectRoot string) (int, error) {
	findings, err := Load(projectRoot)
	if err != nil {
		return 0, err
	}
	now := time.Now().UTC()
	n := 0
	for i := range findings {
		if findings[i].Unread() {
			findings[i].ReadAt = &now
			n++
		}
	}
	if n == 0 {
		return 0, nil
	}
	return n, save(projectRoot, findings)
}

// Drop removes a finding the reader has decided on. Returns false when no
// finding carries that ID, so a caller can say "no such finding" rather than
// silently reporting success.
func Drop(projectRoot, id string) (bool, error) {
	findings, err := Load(projectRoot)
	if err != nil {
		return false, err
	}
	for i := range findings {
		if findings[i].ID == id {
			return true, save(projectRoot, append(findings[:i:i], findings[i+1:]...))
		}
	}
	return false, nil
}

// UnreadCount is what the session signal reports — and all it reports.
func UnreadCount(projectRoot string) int {
	findings, err := Load(projectRoot)
	if err != nil {
		return 0
	}
	n := 0
	for _, f := range findings {
		if f.Unread() {
			n++
		}
	}
	return n
}

func save(projectRoot string, findings []Finding) error {
	p, err := Path(projectRoot)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(file{SchemaVersion: SchemaVersion, Findings: findings}, "", "  ")
	if err != nil {
		return err
	}
	// Write via a temp file in the same directory, so an interrupted write leaves
	// the previous digest intact rather than a truncated one.
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, append(b, '\n'), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}
