package dispatch

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWriteReadRunState(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".delivery", "runstate.json")
	now := time.Date(2026, 7, 4, 12, 0, 0, 0, time.UTC)
	rs := &RunState{
		SprintSlug: "s1",
		Units: map[int]*UnitRunState{
			42: {ID: 42, Branch: "delivery/s1/42", WorktreePath: "/wt/42", PID: 111, Phase: "implement"},
		},
	}
	if err := WriteRunState(path, rs, now); err != nil {
		t.Fatal(err)
	}

	got, err := ReadRunState(path)
	if err != nil {
		t.Fatalf("ReadRunState: %v", err)
	}
	if got.SprintSlug != "s1" || !got.UpdatedAt.Equal(now) {
		t.Errorf("round-trip mismatch: %+v", got)
	}
	u := got.Units[42]
	if u == nil || u.Branch != "delivery/s1/42" || u.PID != 111 {
		t.Errorf("unit 42 round-trip = %+v", u)
	}
}

func TestReadRunStateMissingIsNotExist(t *testing.T) {
	_, err := ReadRunState(filepath.Join(t.TempDir(), "nope.json"))
	if !os.IsNotExist(err) {
		t.Errorf("missing run-state should be os.IsNotExist, got %v", err)
	}
}

func TestObservePhaseResetsOnChange(t *testing.T) {
	t0 := time.Date(2026, 7, 4, 12, 0, 0, 0, time.UTC)
	u := &UnitRunState{ID: 1, Phase: "implement", PhaseEnteredAt: t0}

	// Same phase, later heartbeat: PhaseEnteredAt must NOT move.
	hb1 := t0.Add(1 * time.Minute)
	u.ObservePhase("implement", hb1, t0.Add(1*time.Minute))
	if !u.PhaseEnteredAt.Equal(t0) {
		t.Errorf("PhaseEnteredAt moved on same phase: %v", u.PhaseEnteredAt)
	}
	if !u.LastHeartbeat.Equal(hb1) {
		t.Errorf("heartbeat not updated: %v", u.LastHeartbeat)
	}

	// Phase change: PhaseEnteredAt resets to now (so #12's phase-stall clock
	// starts at the transition).
	t2 := t0.Add(5 * time.Minute)
	u.ObservePhase("self-test", t2, t2)
	if u.Phase != "self-test" || !u.PhaseEnteredAt.Equal(t2) {
		t.Errorf("phase change should reset PhaseEnteredAt to now: %+v", u)
	}
}

func TestRunStatePath(t *testing.T) {
	got := RunStatePath("/proj")
	want := filepath.Join("/proj", ".delivery", "runstate.json")
	if got != want {
		t.Errorf("RunStatePath = %q, want %q", got, want)
	}
}
