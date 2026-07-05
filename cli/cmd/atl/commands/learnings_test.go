package commands

import (
	"testing"

	"github.com/agentteamland/atl/cli/internal/queue"
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
