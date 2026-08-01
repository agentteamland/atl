package drain

import (
	"path/filepath"
	"testing"

	"github.com/agentteamland/atl/cli/internal/queue"
)

// testKnown is the active channel set a caller passes in: the platform's own
// channel plus one a team declared.
var testKnown = []string{"learning", "profile-fact"}

func newStore(t *testing.T) *queue.Store {
	t.Helper()
	s, err := queue.Open(filepath.Join(t.TempDir(), "q.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func TestDrainEnqueues(t *testing.T) {
	s := newStore(t)
	text := "<!-- learning: A --> noise <!-- profile-fact: B -->"

	r, err := Drain(text, "proj", s, testKnown)
	if err != nil {
		t.Fatalf("drain: %v", err)
	}
	if r.Found != 2 || r.Enqueued != 2 {
		t.Fatalf("first drain: found=%d enqueued=%d, want 2/2", r.Found, r.Enqueued)
	}
	if pending, _ := s.Pending("proj", ""); len(pending) != 2 {
		t.Fatalf("want 2 pending, got %d", len(pending))
	}
}

// TestReDrainIdempotent is the load-bearing test: draining the same text twice
// must enqueue nothing the second pass — the re-report bug class is dead.
func TestReDrainIdempotent(t *testing.T) {
	s := newStore(t)
	text := "<!-- learning: same fact -->"

	if _, err := Drain(text, "proj", s, testKnown); err != nil {
		t.Fatalf("first drain: %v", err)
	}
	r, err := Drain(text, "proj", s, testKnown)
	if err != nil {
		t.Fatalf("second drain: %v", err)
	}
	if r.Found != 1 {
		t.Fatalf("second drain still finds the marker: found=%d, want 1", r.Found)
	}
	if r.Enqueued != 0 {
		t.Fatalf("second drain must enqueue 0 (dedup), got %d", r.Enqueued)
	}
	if pending, _ := s.Pending("proj", ""); len(pending) != 1 {
		t.Fatalf("want 1 pending after re-drain, got %d", len(pending))
	}
}

func TestDrainNoMarkers(t *testing.T) {
	s := newStore(t)
	r, err := Drain("prose with nothing to capture", "proj", s, testKnown)
	if err != nil {
		t.Fatalf("drain: %v", err)
	}
	if r.Found != 0 || r.Enqueued != 0 {
		t.Fatalf("want 0/0, got found=%d enqueued=%d", r.Found, r.Enqueued)
	}
}

// The queue has exactly two write seams; this is one of them. An undeclared
// channel must never reach the store — a phantom item sits in the project's
// bucket forever, inflating `learnings status` and doctor's backlog check while
// no drain will ever claim it. The marker LOOKS captured, which is the failure
// mode the declared-set decision exists to prevent.
func TestDrainRefusesAnUndeclaredChannel(t *testing.T) {
	s := newStore(t)
	text := "<!-- learning: kept --> <!-- profile-fact: refused --> <!-- learnin: typo -->"

	// Core-only: no team declares profile-fact here.
	r, err := Drain(text, "proj", s, []string{"learning"})
	if err != nil {
		t.Fatalf("drain: %v", err)
	}
	if r.Found != 1 || r.Enqueued != 1 {
		t.Fatalf("found=%d enqueued=%d, want 1/1 — only the declared channel may be counted", r.Found, r.Enqueued)
	}
	pending, _ := s.Pending("proj", "")
	if len(pending) != 1 {
		t.Fatalf("want 1 pending, got %d — an undeclared channel reached the store", len(pending))
	}
	if pending[0].Channel != queue.ChannelLearning {
		t.Fatalf("queued channel = %q, want learning", pending[0].Channel)
	}

	// The typo is one edit from the ACTIVE channel, so it is reported rather than
	// lost silently. `profile-fact` is not: with profile-team uninstalled it is
	// not a channel here at all, and it is spelled exactly, so it is an ordinary
	// unrecognized comment — reporting it would nag on a machine that simply
	// doesn't have that team.
	if r.NearMiss["learnin"] != 1 {
		t.Errorf("a near-miss of the active channel must be reported, got %v", r.NearMiss)
	}
	if _, reported := r.NearMiss["profile-fact"]; reported {
		t.Errorf("an exactly-spelled undeclared channel is not a typo — must not be reported: %v", r.NearMiss)
	}
}
