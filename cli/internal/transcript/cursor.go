package transcript

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/agentteamland/atl/cli/internal/scope"
)

// CursorSchemaVersion is the on-disk schema version of the mine cursor.
const CursorSchemaVersion = 1

// Cursor records how far each capture channel has mined each of a project's
// transcripts — the state that turns the mine from a backward tail into a
// forward sweep.
//
// Without it the mine reads the most recent N bytes of prose on every run, so
// whatever accumulated between one run's window and the next run's window is
// read by neither and nothing records that it was skipped. Content-hash dedup in
// the queue makes RE-reading safe; it does nothing for never-reading, and those
// are different failures. The cursor closes the second one: each run resumes
// where the last stopped, and what does not fit in one run's budget is reported
// as a backlog instead of vanishing.
//
// It is keyed per CHANNEL because the mine has more than one consumer — /drain
// sweeps for `learning`, /profile-drain for `profile-fact`, and both can run off
// the same turn. A single shared cursor would let whichever ran first consume
// the window the other still needed, manufacturing on one channel exactly the
// silent loss this fixes on the other.
type Cursor struct {
	SchemaVersion int `json:"schemaVersion"`
	// Channels maps a channel name to that channel's per-transcript byte
	// offsets, keyed by the transcript's base name (the session UUID). Absent
	// channel = never mined; absent file within a known channel = a transcript
	// that appeared after the channel's baseline, so it is mined from the start.
	Channels map[string]map[string]int64 `json:"channels"`
}

// CursorPath returns the cursor file for a project: ~/.atl/mine-cursor/<hash>.json.
// Namespaced per project because transcripts are (one directory per project
// slug) — a shared file would let one project's mine advance another's.
func CursorPath(projectRoot string) (string, error) {
	dir, err := scope.LayerDir(scope.Global, "")
	if err != nil {
		return "", err
	}
	sum := sha1.Sum([]byte(filepath.Clean(projectRoot)))
	return filepath.Join(dir, "mine-cursor", hex.EncodeToString(sum[:])[:16]+".json"), nil
}

// LoadCursor reads a project's cursor. A missing or unreadable file yields an
// empty cursor rather than an error: the mine is a best-effort recovery path, and
// an unreadable cursor must degrade to "never mined" (re-read, safe) rather than
// fail the drain.
func LoadCursor(projectRoot string) (*Cursor, error) {
	p, err := CursorPath(projectRoot)
	if err != nil {
		return nil, err
	}
	c := &Cursor{SchemaVersion: CursorSchemaVersion, Channels: map[string]map[string]int64{}}
	b, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return c, nil
		}
		return nil, err
	}
	if err := json.Unmarshal(b, c); err != nil {
		return &Cursor{SchemaVersion: CursorSchemaVersion, Channels: map[string]map[string]int64{}}, nil
	}
	if c.Channels == nil {
		c.Channels = map[string]map[string]int64{}
	}
	c.SchemaVersion = CursorSchemaVersion
	return c, nil
}

// Save atomically persists the cursor (tmp + rename), so a drain that dies
// mid-write leaves the previous cursor intact rather than a truncated one.
func (c *Cursor) Save(projectRoot string) error {
	p, err := CursorPath(projectRoot)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	c.SchemaVersion = CursorSchemaVersion
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

// Known reports whether this channel has ever been mined. The first run for a
// channel has no baseline, so it reads the recent tail (the behavior it had
// before the cursor existed) and baselines every transcript at its current end,
// rather than replaying every session the project has ever had.
func (c *Cursor) Known(channel string) bool {
	_, ok := c.Channels[channel]
	return ok
}

// Offset returns how far this channel has mined the named transcript, and
// whether an offset was recorded at all. A transcript with no entry in a known
// channel is new since the baseline and is mined from 0.
func (c *Cursor) Offset(channel, file string) (int64, bool) {
	off, ok := c.Channels[channel][file]
	return off, ok
}

// SetOffset records a channel's mined-to offset for one transcript.
func (c *Cursor) SetOffset(channel, file string, off int64) {
	if c.Channels == nil {
		c.Channels = map[string]map[string]int64{}
	}
	if c.Channels[channel] == nil {
		c.Channels[channel] = map[string]int64{}
	}
	c.Channels[channel][file] = off
}

// Prune drops recorded files that no longer exist, so a cursor cannot grow
// without bound as sessions are cleaned up. keep holds the base names currently
// on disk.
func (c *Cursor) Prune(channel string, keep map[string]bool) {
	for file := range c.Channels[channel] {
		if !keep[file] {
			delete(c.Channels[channel], file)
		}
	}
}
