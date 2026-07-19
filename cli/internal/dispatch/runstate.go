package dispatch

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// RunState is the supervisor's durable mirror of in-flight work (#11/#12): the
// RUNNING units (which unit is running where, since when, the last phase observed)
// plus the ids that reached DONE. It is persisted via the canonical tmp+rename write.
//
// The RUNNING set is an OBSERVABILITY snapshot: restart reconciliation of in-flight
// worktrees is git-based (Run -> Worktree.Reconcile applies the branch-hygiene
// asymmetry to leftover delivery/* worktrees), because durable git state — not a
// supervisor mirror — is what a fresh process trusts for un-integrated work.
//
// The DONE set is the one genuinely recovery-relevant field, and it exists because
// git CANNOT recover done-ness once a completed unit's branch is torn down (Teardown
// deletes it locally + on origin, and plan.json stores no tip SHA), and the
// zero-backend engine cannot ask the tracker. So the ids that reached done are
// checkpointed here and re-seeded at restart (markPreviouslyDone) so a restart never
// re-runs an already-merged, already-closed unit (which would re-open a Done item and
// open a duplicate PR). Guarded by SprintSlug so a stale/other-sprint file can never
// wrongly skip work. Zero LLM context — pure run-state, never a conversation.
type RunState struct {
	SprintSlug string                `json:"sprintSlug"`
	Units      map[int]*UnitRunState `json:"units"`          // keyed by work-item id — the RUNNING set (observability)
	Done       []int                 `json:"done,omitempty"` // ids that reached stateDone — the restart-idempotency substrate
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
