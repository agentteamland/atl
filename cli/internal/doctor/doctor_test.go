package doctor

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/agentteamland/atl/cli/internal/queue"
)

func newStore(t *testing.T) *queue.Store {
	t.Helper()
	s, err := queue.Open(filepath.Join(t.TempDir(), "q.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func TestBacklogCheck(t *testing.T) {
	s := newStore(t)

	if r := backlogCheck(s, "p"); r.Status != OK {
		t.Fatalf("empty backlog should be OK, got %v (%s)", r.Status, r.Detail)
	}
	for i := 0; i <= BacklogWarn; i++ {
		if _, err := s.Enqueue("p", queue.Item{ID: fmt.Sprintf("i%d", i), Channel: queue.ChannelLearning, Payload: "x"}); err != nil {
			t.Fatal(err)
		}
	}
	if r := backlogCheck(s, "p"); r.Status != Warn {
		t.Fatalf("over-threshold backlog should Warn, got %v", r.Status)
	}
}

func TestCursorCheck(t *testing.T) {
	s := newStore(t)
	now := time.Unix(1_000_000, 0).UTC()

	// never ticked, nothing queued → OK
	if r := cursorCheck(s, "p", now); r.Status != OK {
		t.Fatalf("zero last-tick, no items: want OK, got %v (%s)", r.Status, r.Detail)
	}
	// never ticked, items queued → Warn
	if _, err := s.Enqueue("p", queue.Item{ID: "x", Channel: queue.ChannelLearning, Payload: "p"}); err != nil {
		t.Fatal(err)
	}
	if r := cursorCheck(s, "p", now); r.Status != Warn {
		t.Fatalf("zero last-tick with items: want Warn, got %v", r.Status)
	}
	// freshness is judged from the last-tick wall-clock, not the transcript cursor:
	// a stale transcript-mtime cursor must NOT false-warn after a fresh tick.
	if err := s.SetCursor("p", now.Add(-48*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := s.SetLastTick("p", now.Add(-time.Minute)); err != nil {
		t.Fatal(err)
	}
	if r := cursorCheck(s, "p", now); r.Status != OK {
		t.Fatalf("fresh last-tick (old cursor): want OK, got %v (%s)", r.Status, r.Detail)
	}
	// stale last-tick with items → Warn
	if err := s.SetLastTick("p", now.Add(-48*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if r := cursorCheck(s, "p", now); r.Status != Warn {
		t.Fatalf("stale last-tick with items: want Warn, got %v", r.Status)
	}
}

func TestWorst(t *testing.T) {
	if Worst(nil) != OK {
		t.Fatal("empty results should be OK")
	}
	got := Worst([]Result{{Status: OK}, {Status: Warn}, {Status: OK}})
	if got != Warn {
		t.Fatalf("want Warn, got %v", got)
	}
	got = Worst([]Result{{Status: Warn}, {Status: Fail}})
	if got != Fail {
		t.Fatalf("want Fail, got %v", got)
	}
}
