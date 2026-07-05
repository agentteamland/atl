package queue

import (
	"path/filepath"
	"testing"
	"time"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := Open(filepath.Join(t.TempDir(), "queue.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

// TestEnqueueDedup is the load-bearing test: the same marker transferred twice
// must dedup, so transfer is exactly-once and re-report is impossible.
func TestEnqueueDedup(t *testing.T) {
	s := newTestStore(t)
	it := Item{
		ID:         NewID(ChannelLearning, "fact A"),
		Channel:    ChannelLearning,
		Payload:    "fact A",
		EnqueuedAt: time.Unix(1, 0).UTC(),
	}

	added, err := s.Enqueue("proj", it)
	if err != nil || !added {
		t.Fatalf("first enqueue: added=%v err=%v", added, err)
	}
	added, err = s.Enqueue("proj", it)
	if err != nil {
		t.Fatalf("second enqueue err: %v", err)
	}
	if added {
		t.Fatal("second enqueue of same ID must dedup (added=false)")
	}
	pending, err := s.Pending("proj", "")
	if err != nil {
		t.Fatalf("pending: %v", err)
	}
	if len(pending) != 1 {
		t.Fatalf("want 1 pending after dedup, got %d", len(pending))
	}
}

// TestProcessedThenDeleted asserts deletion frees the pending queue.
func TestProcessedThenDeleted(t *testing.T) {
	s := newTestStore(t)
	it := Item{ID: NewID(ChannelLearning, "fact B"), Channel: ChannelLearning, Payload: "fact B"}
	if _, err := s.Enqueue("proj", it); err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	if err := s.Delete("proj", it.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if pending, _ := s.Pending("proj", ""); len(pending) != 0 {
		t.Fatalf("want 0 pending after delete, got %d", len(pending))
	}
	// Deleting again is a no-op, not an error.
	if err := s.Delete("proj", it.ID); err != nil {
		t.Fatalf("re-delete should be a no-op: %v", err)
	}
}

// TestAckedMarkerDoesNotReReport is the re-report regression test: once a marker
// is acked (Delete), re-enqueuing the same marker — as a transcript re-scan does
// after the coarse modtime cursor lets a still-growing session file through —
// must be a dedup no-op. Before the processed-set tombstone, ack deleted the
// pending item and the next tick re-added it (the confirmed live re-report bug).
func TestAckedMarkerDoesNotReReport(t *testing.T) {
	s := newTestStore(t)
	it := Item{ID: NewID(ChannelLearning, "drained fact"), Channel: ChannelLearning, Payload: "drained fact"}

	if added, err := s.Enqueue("proj", it); err != nil || !added {
		t.Fatalf("first enqueue: added=%v err=%v", added, err)
	}
	if err := s.Delete("proj", it.ID); err != nil { // ack / drain
		t.Fatalf("ack: %v", err)
	}

	// The next tick re-scans the transcript and re-enqueues the same marker.
	added, err := s.Enqueue("proj", it)
	if err != nil {
		t.Fatalf("re-enqueue err: %v", err)
	}
	if added {
		t.Fatal("re-enqueue of an acked marker must dedup against the tombstone (added=false)")
	}
	if pending, _ := s.Pending("proj", ""); len(pending) != 0 {
		t.Fatalf("acked marker re-reported: want 0 pending, got %d", len(pending))
	}
}

// TestTombstoneIsPerProject proves the processed-set is scoped per project — an
// acked ID in one project must not suppress the same marker in another.
func TestTombstoneIsPerProject(t *testing.T) {
	s := newTestStore(t)
	it := Item{ID: NewID(ChannelLearning, "shared fact"), Channel: ChannelLearning, Payload: "shared fact"}

	if _, err := s.Enqueue("projA", it); err != nil {
		t.Fatalf("enqueue A: %v", err)
	}
	if err := s.Delete("projA", it.ID); err != nil {
		t.Fatalf("ack A: %v", err)
	}
	added, err := s.Enqueue("projB", it)
	if err != nil || !added {
		t.Fatalf("projB must not inherit projA's tombstone: added=%v err=%v", added, err)
	}
	if pending, _ := s.Pending("projB", ""); len(pending) != 1 {
		t.Fatalf("want 1 pending in projB, got %d", len(pending))
	}
}

// TestPerProjectIsolation proves the same ID in two projects does not collide.
func TestPerProjectIsolation(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.Enqueue("projA", Item{ID: "1", Channel: ChannelLearning, Payload: "a"}); err != nil {
		t.Fatalf("enqueue A: %v", err)
	}
	if _, err := s.Enqueue("projB", Item{ID: "1", Channel: ChannelProfileFact, Payload: "b"}); err != nil {
		t.Fatalf("enqueue B: %v", err)
	}

	a, _ := s.Pending("projA", "")
	b, _ := s.Pending("projB", "")
	if len(a) != 1 || len(b) != 1 {
		t.Fatalf("isolation broken: A=%d B=%d", len(a), len(b))
	}
	if a[0].Payload != "a" || b[0].Payload != "b" {
		t.Fatalf("cross-project leak: A=%q B=%q", a[0].Payload, b[0].Payload)
	}
}

// TestChannelFilterAndCounts covers the generic multi-channel surface that
// `atl learnings status` reads.
func TestChannelFilterAndCounts(t *testing.T) {
	s := newTestStore(t)
	for _, it := range []Item{
		{ID: "l1", Channel: ChannelLearning, Payload: "x"},
		{ID: "l2", Channel: ChannelLearning, Payload: "y"},
		{ID: "pf1", Channel: ChannelProfileFact, Payload: "z"},
	} {
		if _, err := s.Enqueue("p", it); err != nil {
			t.Fatalf("enqueue %s: %v", it.ID, err)
		}
	}

	learnings, _ := s.Pending("p", ChannelLearning)
	if len(learnings) != 2 {
		t.Fatalf("want 2 learning items, got %d", len(learnings))
	}
	counts, _ := s.Counts("p")
	if counts[ChannelLearning] != 2 || counts[ChannelProfileFact] != 1 {
		t.Fatalf("counts wrong: %+v", counts)
	}
}

func TestEnqueueValidation(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.Enqueue("p", Item{Channel: ChannelLearning, Payload: "x"}); err == nil {
		t.Fatal("empty ID should be rejected")
	}
	if _, err := s.Enqueue("p", Item{ID: "x", Payload: "x"}); err == nil {
		t.Fatal("empty channel should be rejected")
	}
}

func TestCursor(t *testing.T) {
	s := newTestStore(t)

	c, err := s.Cursor("proj")
	if err != nil {
		t.Fatalf("cursor: %v", err)
	}
	if !c.IsZero() {
		t.Fatalf("default cursor should be zero, got %v", c)
	}

	now := time.Unix(1000, 0).UTC()
	if err := s.SetCursor("proj", now); err != nil {
		t.Fatalf("set cursor: %v", err)
	}
	got, _ := s.Cursor("proj")
	if !got.Equal(now) {
		t.Fatalf("cursor: got %v want %v", got, now)
	}

	// the cursor bucket must not leak into item listings
	if _, err := s.Enqueue("proj", Item{ID: "x", Channel: ChannelLearning, Payload: "p"}); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	if pending, _ := s.Pending("proj", ""); len(pending) != 1 {
		t.Fatalf("cursor leaked into pending or item lost: %d items", len(pending))
	}
}

// TestPendingOrdering asserts stable EnqueuedAt-then-ID ordering.
func TestPendingOrdering(t *testing.T) {
	s := newTestStore(t)
	_, _ = s.Enqueue("p", Item{ID: "b", Channel: ChannelLearning, Payload: "2", EnqueuedAt: time.Unix(2, 0).UTC()})
	_, _ = s.Enqueue("p", Item{ID: "a", Channel: ChannelLearning, Payload: "1", EnqueuedAt: time.Unix(1, 0).UTC()})
	_, _ = s.Enqueue("p", Item{ID: "c", Channel: ChannelLearning, Payload: "3", EnqueuedAt: time.Unix(3, 0).UTC()})

	pending, _ := s.Pending("p", "")
	got := []string{pending[0].Payload, pending[1].Payload, pending[2].Payload}
	want := []string{"1", "2", "3"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("ordering: got %v want %v", got, want)
		}
	}
}
