package marker

import "testing"

func TestParse(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []Marker
	}{
		{
			name: "single learning marker",
			in:   "blah blah <!-- learning: prefers Node for APIs --> trailing",
			want: []Marker{{Channel: "learning", Body: "prefers Node for APIs"}},
		},
		{
			name: "multi-line profile-fact",
			in:   "text\n<!-- profile-fact:\n  entity: ahmet\n  field: traits.fears\n-->\nmore",
			want: []Marker{{Channel: "profile-fact", Body: "entity: ahmet\n  field: traits.fears"}},
		},
		{
			name: "multiple markers in order",
			in:   "<!-- learning: A --> mid <!-- profile-fact: B -->",
			want: []Marker{{Channel: "learning", Body: "A"}, {Channel: "profile-fact", Body: "B"}},
		},
		{
			name: "adjacent markers do not merge (non-greedy)",
			in:   "<!-- learning: first --><!-- learning: second -->",
			want: []Marker{{Channel: "learning", Body: "first"}, {Channel: "learning", Body: "second"}},
		},
		{
			// An unclosed marker must be discarded on its own, not swallow the next
			// marker's close (which would garble the first and drop the second).
			name: "unclosed marker does not swallow the following one",
			in:   "<!-- learning: truncated\n<!-- learning: intact -->",
			want: []Marker{{Channel: "learning", Body: "intact"}},
		},
		{
			name: "unknown channel ignored",
			in:   "<!-- todo: not ours --> <!-- learning: ours -->",
			want: []Marker{{Channel: "learning", Body: "ours"}},
		},
		{
			name: "empty body ignored",
			in:   "<!-- learning:   --> <!-- learning: real -->",
			want: []Marker{{Channel: "learning", Body: "real"}},
		},
		{
			name: "no markers",
			in:   "just some prose with no markers at all",
			want: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := Parse(tc.in)
			if len(got) != len(tc.want) {
				t.Fatalf("got %d markers, want %d: %+v", len(got), len(tc.want), got)
			}
			for i := range tc.want {
				if got[i] != tc.want[i] {
					t.Errorf("marker %d: got %+v, want %+v", i, got[i], tc.want[i])
				}
			}
		})
	}
}

// TestHasIsChannelScoped is the regression for the capture-watchdog's
// channel-blindness: the watchdog used a channel-agnostic `len(Parse(s)) > 0`
// predicate, so ANY marker reset its dry stretch. A session emitting learning
// markers regularly could therefore never trip the watchdog for a missed
// profile-fact — the exact omission it exists to detect, on one of the two
// channels it protects.
func TestHasIsChannelScoped(t *testing.T) {
	learningOnly := "some reply\n<!-- learning: pools are shared, not per-request -->\nmore text"

	// The old, channel-blind predicate is true here — which is the bug.
	if len(Parse(learningOnly)) == 0 {
		t.Fatal("fixture should carry a marker")
	}
	if !Has(learningOnly, "learning") {
		t.Error("Has(learning) = false on text with a learning marker")
	}
	if Has(learningOnly, "profile-fact") {
		t.Error("Has(profile-fact) = true on text with only a learning marker — a learning marker must not mask a missed profile-fact")
	}

	both := learningOnly + "\n<!-- profile-fact:\n  entity: ahmet\n  fields:\n    identity.name: Ahmet\n-->"
	if !Has(both, "learning") || !Has(both, "profile-fact") {
		t.Error("Has should report both channels when both are present")
	}

	if Has("plain text, no markers", "learning") {
		t.Error("Has = true on text with no markers")
	}
	if Has("<!-- unknown-channel: body -->", "unknown-channel") {
		t.Error("Has = true for an unrecognized channel — Parse must gate it")
	}
}
