package dispatch

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// RunState is the supervisor's durable mirror of in-flight work (#11/#12): which
// unit is running where, since when, and the last phase the supervisor observed.
// It is persisted via the canonical tmp+rename atomic write.
//
// It is an OBSERVABILITY snapshot, not the recovery substrate: restart
// reconciliation is worktree/branch-based (Run -> Worktree.Reconcile applies the
// branch-hygiene asymmetry to leftover delivery/* worktrees), because the durable
// git state — not a supervisor-written mirror — is the source of truth a fresh
// process can trust. This file lets a human (or a future status surface) see the
// last in-flight set; ReadRunState exists for that, not for recovery. Zero LLM
// context — pure run-state, never a conversation.
type RunState struct {
	SprintSlug string                `json:"sprintSlug"`
	Units      map[int]*UnitRunState `json:"units"` // keyed by work-item id
	UpdatedAt  time.Time             `json:"updatedAt"`
}

// UnitRunState is one running unit's supervisor-side record.
type UnitRunState struct {
	ID             int       `json:"id"`
	Branch         string    `json:"branch"`
	WorktreePath   string    `json:"worktreePath"`
	Stage          string    `json:"stage"` // current pipeline stage: developer | tester | tech-lead
	PID            int       `json:"pid"`
	CreatedAt      time.Time `json:"createdAt"`
	Phase          string    `json:"phase"`          // last observed phase (from status.json)
	PhaseEnteredAt time.Time `json:"phaseEnteredAt"` // when the supervisor first saw this phase — drives phase-stall (#12)
	LastHeartbeat  time.Time `json:"lastHeartbeat"`
}

// ObservePhase folds a fresh status.json reading into the unit's run-state,
// resetting PhaseEnteredAt when the phase changes so #12's phase-stall clock
// starts at the transition, not at spawn.
func (u *UnitRunState) ObservePhase(phase string, heartbeat, now time.Time) {
	if phase != u.Phase {
		u.Phase = phase
		u.PhaseEnteredAt = now
	}
	u.LastHeartbeat = heartbeat
}

// RunStatePath returns the canonical run-state location for a project:
// <projectRoot>/.delivery/runstate.json — supervisor-written, transient,
// alongside the skill-written plan.json.
func RunStatePath(projectRoot string) string {
	return filepath.Join(projectRoot, ".delivery", "runstate.json")
}

// WriteRunState atomically writes rs to path (tmp + rename; the manifest.go
// idiom). UpdatedAt is stamped by the caller-supplied clock.
func WriteRunState(path string, rs *RunState, now time.Time) error {
	rs.UpdatedAt = now
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(rs, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(b, '\n'), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// ReadRunState loads run-state from path. A missing file is the expected
// first-run / clean-restart state — callers can check os.IsNotExist.
func ReadRunState(path string) (*RunState, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var rs RunState
	if err := json.Unmarshal(b, &rs); err != nil {
		return nil, fmt.Errorf("parse run-state %s: %w", path, err)
	}
	return &rs, nil
}
