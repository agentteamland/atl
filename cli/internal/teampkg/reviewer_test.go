package teampkg

import (
	"encoding/json"
	"testing"
)

func withCapabilities(t *testing.T, raw string) *TeamManifest {
	t.Helper()
	var tm TeamManifest
	if err := json.Unmarshal([]byte(raw), &tm); err != nil {
		t.Fatalf("fixture is not valid team.json: %v", err)
	}
	return &tm
}

// The shape delivery-team actually ships: `review` IS a capability whose value
// is the agent name, a bare string — not a field inside a capability object the
// way `store` and `channel` are.
func TestDeclaredReviewerReadsTheBareString(t *testing.T) {
	tm := withCapabilities(t, `{"capabilities":{"review":"tech-lead"}}`)
	if got := tm.DeclaredReviewer(); got != "tech-lead" {
		t.Errorf("DeclaredReviewer() = %q, want %q", got, "tech-lead")
	}
}

// A team declaring none is the common case and must be silent, not an error:
// /create-pr skips it and falls back to its generic reviewer.
func TestDeclaredReviewerIsEmptyWhenUndeclared(t *testing.T) {
	for name, raw := range map[string]string{
		"no capabilities at all": `{}`,
		"empty capabilities":     `{"capabilities":{}}`,
		"a different capability": `{"capabilities":{"profile":{"role":"provider"}}}`,
	} {
		if got := withCapabilities(t, raw).DeclaredReviewer(); got != "" {
			t.Errorf("%s: DeclaredReviewer() = %q, want empty", name, got)
		}
	}
}

// Malformed declarations are tolerated rather than fatal — the same posture as
// DeclaredStores and DeclaredChannels. A team.json that gets this wrong must not
// break install; it simply declares no reviewer.
func TestDeclaredReviewerToleratesAWrongShape(t *testing.T) {
	for name, raw := range map[string]string{
		"an object":     `{"capabilities":{"review":{"agent":"tech-lead"}}}`,
		"a list":        `{"capabilities":{"review":["tech-lead"]}}`,
		"a number":      `{"capabilities":{"review":7}}`,
		"null":          `{"capabilities":{"review":null}}`,
		"blank padding": `{"capabilities":{"review":"   "}}`,
	} {
		if got := withCapabilities(t, raw).DeclaredReviewer(); got != "" {
			t.Errorf("%s: DeclaredReviewer() = %q, want empty", name, got)
		}
	}
}

// Whitespace is trimmed, so a hand-edited team.json cannot produce an agent name
// that silently fails to resolve.
func TestDeclaredReviewerTrims(t *testing.T) {
	tm := withCapabilities(t, `{"capabilities":{"review":"  tech-lead\n"}}`)
	if got := tm.DeclaredReviewer(); got != "tech-lead" {
		t.Errorf("DeclaredReviewer() = %q, want %q", got, "tech-lead")
	}
}

// The regression this whole change exists for: the reviewer must survive into
// the INSTALLED manifest. team.json is not an installable asset, so a consumer
// that only sees the installed tree has no other way to learn the declaration —
// which is exactly why /create-pr step 5b silently did nothing.
func TestTheShippedTeamManifestStillDeclaresItsReviewer(t *testing.T) {
	tm, err := ReadManifest("../../../teams/delivery-team")
	if err != nil {
		t.Fatalf("read delivery-team's team.json: %v", err)
	}
	if got := tm.DeclaredReviewer(); got == "" {
		t.Fatal("delivery-team declares capabilities.review but DeclaredReviewer() read nothing")
	}
}
