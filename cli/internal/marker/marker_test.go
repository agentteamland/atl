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
