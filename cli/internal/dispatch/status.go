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
	HeartbeatTS       time.Time `json:"heartbeatTs"`       // last heartbeat — written on every phase change + a ~30-60s tick
	Blocker           string    `json:"blocker"`           // non-empty when the worker reported a blocker
	LastOutputSummary string    `json:"lastOutputSummary"` // short human-readable last-progress line
}

// StatusFileName is the fixed name a worker writes inside its worktree.
const StatusFileName = "status.json"

// ReadStatus reads and parses <worktree>/status.json. A missing file is a
// distinct, expected pre-first-heartbeat state — callers can check
// os.IsNotExist on the returned error.
func ReadStatus(worktreePath string) (*Status, error) {
	b, err := os.ReadFile(filepath.Join(worktreePath, StatusFileName))
	if err != nil {
		return nil, err
	}
	var s Status
	if err := json.Unmarshal(b, &s); err != nil {
		return nil, fmt.Errorf("parse status %s: %w", worktreePath, err)
	}
	return &s, nil
}
