package marker

import "testing"

// testKnown is the active-channel set most cases run against: the platform's own
// channel plus one a team declared. The set is injected now — this package holds
// no allowlist of its own.
var testKnown = []string{"learning", "profile-fact"}

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
			got := Parse(tc.in, testKnown)
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
	if len(Parse(learningOnly, testKnown)) == 0 {
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
	// Has answers "does this text carry a marker on THIS channel", and the caller
	// only ever asks about a channel it already knows is active (the watchdog
	// iterates the active set). Gating an unknown channel is the caller's job now
	// that the allowlist is injected, so Has reports honestly for any name.
	if !Has("<!-- audit-note: body -->", "audit-note") {
		t.Error("Has must report a marker on the channel it was asked about")
	}
	if Has("<!-- audit-note: body -->", "learning") {
		t.Error("Has must not report a marker that is on a different channel")
	}
}

// The allowlist is the CALLER's, not this package's. A channel is recognized
// exactly when it is in known — so an undeclared team channel is dropped even
// though it was hardcoded as recognized before this became a declaration, and a
// channel this package has never heard of is recognized when a team declares it.
func TestParseHonorsTheInjectedAllowlist(t *testing.T) {
	text := "<!-- learning: kept --> <!-- profile-fact: dropped --> <!-- audit-note: kept too -->"

	// Core-only (no team installed): only the platform's own channel survives.
	got := Parse(text, []string{"learning"})
	if len(got) != 1 || got[0].Channel != "learning" {
		t.Fatalf("core-only allowlist should keep exactly the learning marker, got %+v", got)
	}

	// A channel this package never knew about is recognized once declared.
	got = Parse(text, []string{"learning", "audit-note"})
	if len(got) != 2 || got[1].Channel != "audit-note" {
		t.Fatalf("a declared novel channel must be recognized, got %+v", got)
	}

	// An empty allowlist recognizes nothing — it must not fall back to a builtin set.
	if got := Parse(text, nil); len(got) != 0 {
		t.Fatalf("an empty allowlist must recognize nothing, got %+v", got)
	}
}

// A near-miss is the one unknown channel worth reporting: a marker the agent
// meant to write, mistyped. Everything else — including ATL's own always-loaded
// marker-shaped blocks — must stay silent, or the report false-fires on every
// prompt in every ATL project.
func TestScanReportsOnlyNearMisses(t *testing.T) {
	found, near := Scan("<!-- profile-fct: entity: ahmet -->", testKnown)
	if len(found) != 0 {
		t.Fatalf("a mistyped channel must not be captured, got %+v", found)
	}
	if near["profile-fct"] != 1 {
		t.Errorf("a one-edit typo must be reported as a near-miss, got %v", near)
	}

	// ATL's own always-loaded blocks are marker-SHAPED but not markers.
	for _, noise := range []string{
		"<!-- wiki:index -->",
		"<!-- brainstorm:active:start -->\ncontent\n<!-- brainstorm:active:end -->",
		"<!-- atl:managed -->",
		"<!-- todo: unrelated comment -->",
	} {
		if _, near := Scan(noise, testKnown); len(near) != 0 {
			t.Errorf("%q reported a near-miss (%v) — this fires on every prompt in an ATL project", noise, near)
		}
	}

	// An empty-bodied comment is chrome, not a marker someone meant to write.
	if _, near := Scan("<!-- learnin: -->", testKnown); len(near) != 0 {
		t.Errorf("an empty-bodied near-name must not be reported, got %v", near)
	}

	// Repeats accumulate, so the report can say how much was lost.
	if _, near := Scan("<!-- learnin: a --> <!-- learnin: b -->", testKnown); near["learnin"] != 2 {
		t.Errorf("near-miss count should accumulate, got %v", near)
	}
}

func TestNearlyEqual(t *testing.T) {
	within := [][2]string{
		{"profile-fct", "profile-fact"},   // deletion
		{"profile-facts", "profile-fact"}, // insertion
		{"profile-fadt", "profile-fact"},  // substitution
		{"learning", "learning"},          // identical
		{"learnings", "learning"},
		{"earning", "learning"},
		// Case-sensitive, so one wrong case IS one edit. Harmless in practice:
		// innerRe only matches a lowercase channel, so "Learning:" never parses as
		// a channel at all — this just pins the predicate's stated semantics.
		{"Learning", "learning"},
	}
	for _, p := range within {
		if !NearlyEqual(p[0], p[1]) {
			t.Errorf("NearlyEqual(%q, %q) = false, want true", p[0], p[1])
		}
	}
	beyond := [][2]string{
		{"profile", "profile-fact"},
		{"learn", "learning"},
		{"wiki", "learning"},
		{"", "learning"},
	}
	for _, p := range beyond {
		if NearlyEqual(p[0], p[1]) {
			t.Errorf("NearlyEqual(%q, %q) = true, want false", p[0], p[1])
		}
	}
}
