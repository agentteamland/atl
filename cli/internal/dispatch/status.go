package dispatch

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Status is the live per-worker progress contract (keystone #2) — the four
// fields a `claude -p` worker writes into <worktree>/status.json and the
// supervisor polls. The supervisor treats a worker as alive only while status
// shows BOTH a fresh heartbeat AND forward phase progress (#12). This is the
// only channel from worker to supervisor; Azure is written at durable
// milestones, never polled for liveness.
type Status struct {
	Phase             string    `json:"phase"`             // current worker phase (implement, self-test, pr, …)
	HeartbeatTS       time.Time `json:"heartbeatTs"`       // worker-written, but liveness uses the file mtime instead (ReadStatus overwrites this) — an LLM can't produce a reliable clock value
	Blocker           string    `json:"blocker"`           // non-empty when the worker reported a blocker
	LastOutputSummary string    `json:"lastOutputSummary"` // short human-readable last-progress line
}

// StatusFileName is the fixed name a worker writes inside its worktree.
const StatusFileName = "status.json"

// ReadStatus reads and parses <worktree>/status.json. A missing file is a
// distinct, expected pre-first-heartbeat state — callers can check
// os.IsNotExist on the returned error.
//
// Liveness is measured from the file's MTIME, not the worker-written heartbeatTs:
// a `claude -p` worker is an LLM with no reliable clock and writes a
// placeholder/guessed timestamp (observed in a real run: "…T00:00:00Z"), which
// would make every fresh write look hours stale and trip a false HeartbeatBreach.
// The OS stamps mtime accurately on every write, so "the worker (re)wrote
// status.json" — with any content — IS the heartbeat; the timestamp field is
// advisory. Phase progress still comes from the parsed Phase field, which the
// worker sets correctly. So we overwrite HeartbeatTS with the mtime here.
func ReadStatus(worktreePath string) (*Status, error) {
	path := filepath.Join(worktreePath, StatusFileName)
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var s Status
	if err := json.Unmarshal(b, &s); err != nil {
		return nil, fmt.Errorf("parse status %s: %w", worktreePath, err)
	}
	if fi, statErr := os.Stat(path); statErr == nil {
		s.HeartbeatTS = fi.ModTime()
	}
	return &s, nil
}
