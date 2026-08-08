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
	if _, err := s.Enqueue("projB", Item{ID: "1", Channel: Channel("profile-fact"), Payload: "b"}); err != nil {
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
		{ID: "pf1", Channel: Channel("profile-fact"), Payload: "z"},
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
	if counts[ChannelLearning] != 2 || counts[Channel("profile-fact")] != 1 {
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

// TestWatchdogLatch guards the fire-once contract: the latch round-trips per
// (project, session), distinct projects never see each other's latch, and —
// the ping-pong regression — two concurrent sessions in ONE project each keep
// their own latch, so alternating "newest transcript" between them can never
// re-fire an already-fired stretch.
func TestWatchdogLatch(t *testing.T) {
	s := newTestStore(t)

	const l, pf = "learning", "profile-fact"

	if k, err := s.WatchdogLatch("/p/a", "s1.jsonl", l); err != nil || k != "" {
		t.Fatalf("fresh latch = (%q, %v), want empty", k, err)
	}
	if err := s.SetWatchdogLatch("/p/a", "s1.jsonl", l, "s1.jsonl:3"); err != nil {
		t.Fatal(err)
	}
	if k, _ := s.WatchdogLatch("/p/a", "s1.jsonl", l); k != "s1.jsonl:3" {
		t.Errorf("latch = %q, want s1.jsonl:3", k)
	}
	// Per-project isolation.
	if k, _ := s.WatchdogLatch("/p/b", "s1.jsonl", l); k != "" {
		t.Errorf("project b latch = %q, want empty (no cross-project bleed)", k)
	}
	// Ping-pong regression: a second live session in the SAME project fires for
	// its own stretch without clobbering the first session's latch.
	if err := s.SetWatchdogLatch("/p/a", "s2.jsonl", l, "s2.jsonl:0"); err != nil {
		t.Fatal(err)
	}
	if k, _ := s.WatchdogLatch("/p/a", "s1.jsonl", l); k != "s1.jsonl:3" {
		t.Errorf("s1 latch = %q after s2 fired — a per-project single slot would re-fire s1's stretch", k)
	}
	if k, _ := s.WatchdogLatch("/p/a", "s2.jsonl", l); k != "s2.jsonl:0" {
		t.Errorf("s2 latch = %q, want s2.jsonl:0", k)
	}
	// Per-CHANNEL isolation — the same ping-pong one level down. The two channels
	// measure separate stretches in the same session; a shared slot would let each
	// firing clobber the other's latch and nag on every alternation.
	if k, _ := s.WatchdogLatch("/p/a", "s1.jsonl", pf); k != "" {
		t.Errorf("profile-fact latch = %q, want empty — the learning latch must not answer for it", k)
	}
	if err := s.SetWatchdogLatch("/p/a", "s1.jsonl", pf, "s1.jsonl:0"); err != nil {
		t.Fatal(err)
	}
	if k, _ := s.WatchdogLatch("/p/a", "s1.jsonl", l); k != "s1.jsonl:3" {
		t.Errorf("learning latch = %q after profile-fact fired — a shared slot would re-fire an already-fired stretch", k)
	}
	if k, _ := s.WatchdogLatch("/p/a", "s1.jsonl", pf); k != "s1.jsonl:0" {
		t.Errorf("profile-fact latch = %q, want s1.jsonl:0", k)
	}
	// Overwrite = the new stretch replaces the old for that session+channel.
	if err := s.SetWatchdogLatch("/p/a", "s1.jsonl", l, "s1.jsonl:4"); err != nil {
		t.Fatal(err)
	}
	if k, _ := s.WatchdogLatch("/p/a", "s1.jsonl", l); k != "s1.jsonl:4" {
		t.Errorf("latch = %q, want s1.jsonl:4", k)
	}
	if k, _ := s.WatchdogLatch("/p/a", "s1.jsonl", pf); k != "s1.jsonl:0" {
		t.Errorf("profile-fact latch = %q after learning overwrote — channels must not share a slot", k)
	}
}

// Projects is the only cross-project read in this package, and it exists so a
// bucket whose directory is gone can still be NAMED. Every other surface is
// project-scoped, which is why 13 stranded items produced no signal for three
// weeks.
func TestProjectsSeesEveryBucketIncludingVanishedOnes(t *testing.T) {
	st := newTestStore(t)
	live := t.TempDir()
	gone := filepath.Join(t.TempDir(), "deleted-worktree") // never created

	for _, p := range []string{live, gone} {
		if _, err := st.Enqueue(p, Item{ID: NewID(ChannelLearning, p), Channel: ChannelLearning, Payload: "x"}); err != nil {
			t.Fatal(err)
		}
	}

	got, err := st.Projects()
	if err != nil {
		t.Fatalf("Projects: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d project(s), want both the live and the vanished one: %v", len(got), got)
	}
	if got[gone][ChannelLearning] != 1 {
		t.Fatalf("the vanished bucket is not reported: %v", got)
	}
}

// Reserved buckets are keyed by their own scheme, not by a project path, so they
// must never appear as projects — otherwise the stranded-bucket check would
// report `__cursors__` as a deleted directory on every machine.
func TestProjectsSkipsReservedBuckets(t *testing.T) {
	st := newTestStore(t)
	p := t.TempDir()
	if err := st.SetCursor(p, time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := st.SetWatchdogLatch(p, "sess", "learning", "k"); err != nil {
		t.Fatal(err)
	}
	got, err := st.Projects()
	if err != nil {
		t.Fatalf("Projects: %v", err)
	}
	for k := range got {
		if k == cursorBucket || k == processedBucket || k == watchdogBucket {
			t.Fatalf("reserved bucket %q reported as a project", k)
		}
	}
}

// Recover is the deliberate re-enqueue a stranded bucket needs: keying by the
// repository root stops NEW losses, but nothing looks for the old addresses
// afterwards, so what is already stranded has to be moved before it is written
// off.
func TestRecoverMovesItemsAndRemovesTheSourceBucket(t *testing.T) {
	s := newTestStore(t)
	gone, target := "/gone/worktree/7", "/live/repo"
	it := Item{ID: NewID(ChannelLearning, "worker learning"), Channel: ChannelLearning, Payload: "worker learning", EnqueuedAt: time.Unix(100, 0).UTC()}
	if _, err := s.Enqueue(gone, it); err != nil {
		t.Fatal(err)
	}

	moved, err := s.Recover(target, []string{gone})
	if err != nil || moved != 1 {
		t.Fatalf("Recover: moved %d, err %v — want 1, nil", moved, err)
	}

	got, err := s.Pending(target, "")
	if err != nil || len(got) != 1 {
		t.Fatalf("target holds %d item(s), err %v — want the rescued one", len(got), err)
	}
	// The payload, id and ORIGINAL capture time survive: this is a rescue, not a
	// re-capture, and a re-stamped time would misdate three-week-old knowledge.
	if got[0].Payload != it.Payload || got[0].ID != it.ID || !got[0].EnqueuedAt.Equal(it.EnqueuedAt) {
		t.Fatalf("item was altered in transit: %+v", got[0])
	}
	// The source bucket is gone, so the doctor's stranded check goes quiet — a
	// warning with no way to clear it is one people stop reading.
	projects, err := s.Projects()
	if err != nil {
		t.Fatal(err)
	}
	if _, still := projects[gone]; still {
		t.Fatalf("source bucket %q still reported: %v", gone, projects)
	}
}

// No tombstone is written for the source. A tombstone means "processed", and a
// stranded item never was — writing one would make the payload permanently
// un-re-enqueueable if the same marker were ever re-mined from a transcript.
func TestRecoverDoesNotTombstoneTheSource(t *testing.T) {
	s := newTestStore(t)
	gone, target := "/gone/x", "/live/y"
	it := Item{ID: NewID(ChannelLearning, "p"), Channel: ChannelLearning, Payload: "p"}
	if _, err := s.Enqueue(gone, it); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Recover(target, []string{gone}); err != nil {
		t.Fatal(err)
	}
	added, err := s.Enqueue(gone, it)
	if err != nil {
		t.Fatal(err)
	}
	if !added {
		t.Fatal("re-enqueue into the source was refused — Recover left a tombstone, so the payload is now unrecoverable there")
	}
}

// An id already present in the target wins: recovering twice, or recovering
// something a human already rescued by hand, must not duplicate it.
func TestRecoverSkipsAnItemTheTargetAlreadyHas(t *testing.T) {
	s := newTestStore(t)
	gone, target := "/gone/z", "/live/z"
	it := Item{ID: NewID(ChannelLearning, "dup"), Channel: ChannelLearning, Payload: "dup"}
	if _, err := s.Enqueue(gone, it); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Enqueue(target, it); err != nil {
		t.Fatal(err)
	}
	moved, err := s.Recover(target, []string{gone})
	if err != nil {
		t.Fatal(err)
	}
	if moved != 0 {
		t.Fatalf("moved %d — the target already had this id, so nothing should move", moved)
	}
	got, _ := s.Pending(target, "")
	if len(got) != 1 {
		t.Fatalf("target holds %d copies — a rescue must not duplicate", len(got))
	}
}
