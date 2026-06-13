package drain

import (
	"path/filepath"
	"testing"

	"github.com/agentteamland/atl/cli/internal/queue"
)

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

	r, err := Drain(text, "proj", s)
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

	if _, err := Drain(text, "proj", s); err != nil {
		t.Fatalf("first drain: %v", err)
	}
	r, err := Drain(text, "proj", s)
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
	r, err := Drain("prose with nothing to capture", "proj", s)
	if err != nil {
		t.Fatalf("drain: %v", err)
	}
	if r.Found != 0 || r.Enqueued != 0 {
		t.Fatalf("want 0/0, got found=%d enqueued=%d", r.Found, r.Enqueued)
	}
}
