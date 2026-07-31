package commands

import (
	"strings"
	"testing"

	"github.com/agentteamland/atl/cli/internal/queue"
	"github.com/agentteamland/atl/cli/internal/transcript"
)

func TestFirstLine(t *testing.T) {
	if got := firstLine("one line"); got != "one line" {
		t.Errorf("single line = %q", got)
	}
	if got := firstLine("first\nsecond\nthird"); got != "first …" {
		t.Errorf("multiline = %q, want \"first …\"", got)
	}
	if got := firstLine(""); got != "" {
		t.Errorf("empty = %q", got)
	}
}

func TestStatusJSON(t *testing.T) {
	cases := []struct {
		name   string
		counts map[queue.Channel]int
		want   string
	}{
		// Empty must be an object, "{}", not JSON null — so tooling always
		// gets a channel→count map even when nothing is pending.
		{"nil map", nil, "{}"},
		{"empty map", map[queue.Channel]int{}, "{}"},
		// Multi-channel: string-kinded keys sort deterministically, so the
		// output is stable ("learning" before "profile-fact").
		{
			"multi-channel",
			map[queue.Channel]int{queue.ChannelLearning: 2, queue.ChannelProfileFact: 1},
			`{"learning":2,"profile-fact":1}`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := statusJSON(tc.counts)
			if err != nil {
				t.Fatalf("statusJSON(%v) error: %v", tc.counts, err)
			}
			if got != tc.want {
				t.Errorf("statusJSON(%v) = %q, want %q", tc.counts, got, tc.want)
			}
		})
	}
}

// TestTailByBytes guards the mining surface's only size bound. Before it, the
// command's sole limit was a file COUNT, so the default two-transcript read could
// hand the drain ~900 KB of prose — past a subagent's context, and diluting the
// step exactly when the session was long enough to have forgotten something.
func TestTailByBytes(t *testing.T) {
	turn := func(n int) transcript.Turn { return transcript.Turn{Role: "user", Text: strings.Repeat("x", n)} }

	// Under the cap: everything is emitted, and nothing is claimed to be cut.
	all := []transcript.Turn{turn(10), turn(10)}
	if kept, truncated := tailByBytes(all, 100); len(kept) != 2 || truncated {
		t.Errorf("under cap: kept %d truncated=%v, want 2 false", len(kept), truncated)
	}

	// Over the cap: the OLDEST turns are the ones dropped — a correction the agent
	// missed is most likely in the stretch just gone by.
	five := []transcript.Turn{turn(10), turn(10), turn(10), turn(10), turn(10)}
	kept, truncated := tailByBytes(five, 25)
	if !truncated {
		t.Error("over cap: truncation must be reported so the mine is honest about being partial")
	}
	if len(kept) != 2 {
		t.Fatalf("over cap: kept %d turns, want 2 (25-byte budget, 10 bytes each)", len(kept))
	}
	total := 0
	for _, k := range kept {
		total += len(k.Text)
	}
	if total > 25 {
		t.Errorf("kept %d bytes, over the 25-byte budget", total)
	}

	// A single turn bigger than the whole budget is still emitted: blanking the
	// mining input is worse than overshooting. Nothing older was cut, so this is
	// honestly not a truncation.
	if kept, truncated := tailByBytes([]transcript.Turn{turn(500)}, 25); len(kept) != 1 || truncated {
		t.Errorf("oversized lone turn: kept %d truncated=%v, want 1 false", len(kept), truncated)
	}

	// No turns at all (a project with no transcripts) must not report truncation.
	if kept, truncated := tailByBytes(nil, 25); len(kept) != 0 || truncated {
		t.Errorf("empty: kept %d truncated=%v, want 0 false", len(kept), truncated)
	}
}

func TestResolveAckID(t *testing.T) {
	items := []queue.Item{
		{ID: "aa11cccccccccccc", Channel: queue.ChannelLearning},
		{ID: "aa22dddddddddddd", Channel: queue.ChannelLearning},
		{ID: "bb33eeeeeeeeeeee", Channel: queue.ChannelProfileFact},
	}

	// A full id resolves to itself.
	if id, err := resolveAckID(items, "bb33eeeeeeeeeeee"); err != nil || id != "bb33eeeeeeeeeeee" {
		t.Errorf("full id: got (%q, %v), want that id", id, err)
	}
	// An unambiguous prefix (what peek shows) resolves to the one match.
	if id, err := resolveAckID(items, "bb33"); err != nil || id != "bb33eeeeeeeeeeee" {
		t.Errorf("unambiguous prefix: got (%q, %v), want bb33...", id, err)
	}
	// An ambiguous prefix errors instead of deleting the wrong item.
	if _, err := resolveAckID(items, "aa"); err == nil {
		t.Error("ambiguous prefix should error, matched >1 item")
	}
	// A non-matching id errors instead of a silent no-op.
	if _, err := resolveAckID(items, "zz99"); err == nil {
		t.Error("non-matching id should error, not resolve")
	}
	// An empty/whitespace id errors (never match-all).
	if _, err := resolveAckID(items, "   "); err == nil {
		t.Error("empty id should error")
	}
}
