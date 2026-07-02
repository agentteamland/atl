package pin

import (
	"testing"
)

func TestAddRemoveDup(t *testing.T) {
	s := &Set{}
	if !s.Add("agents/api") {
		t.Error("first Add should report added")
	}
	if s.Add("agents/api") {
		t.Error("duplicate Add should report not-added")
	}
	if len(s.Pins) != 1 {
		t.Errorf("len = %d, want 1", len(s.Pins))
	}
	if !s.Remove("agents/api") {
		t.Error("Remove of present pin should report removed")
	}
	if s.Remove("agents/api") {
		t.Error("Remove of absent pin should report not-removed")
	}
	if len(s.Pins) != 0 {
		t.Errorf("len after remove = %d, want 0", len(s.Pins))
	}
}

func TestPinnedPrefix(t *testing.T) {
	s := &Set{}
	s.Add("agents/backend-agent")
	s.Add("rules/house-style.md")

	pinned := []string{
		"agents/backend-agent",               // exact
		"agents/backend-agent/agent.md",      // nested file
		"agents/backend-agent/children/x.md", // deeper nested
		"rules/house-style.md",           // exact file pin
	}
	for _, p := range pinned {
		if !s.Pinned(p) {
			t.Errorf("Pinned(%q) = false, want true", p)
		}
	}
	notPinned := []string{
		"agents/api",            // prefix-of-pin but not a path boundary (no false match)
		"agents/backend-agent2",     // sibling sharing a string prefix
		"agents/db-agent/x.md",  // unrelated agent
		"rules/house-style.mdx", // not the pinned file
	}
	for _, p := range notPinned {
		if s.Pinned(p) {
			t.Errorf("Pinned(%q) = true, want false", p)
		}
	}
}

func TestNormalize(t *testing.T) {
	cases := map[string]string{
		"  agents/api/  ": "agents/api",
		"./agents/api":    "agents/api",
		"agents/api":      "agents/api",
		"/agents/api":     "agents/api",
	}
	for in, want := range cases {
		if got := Normalize(in); got != want {
			t.Errorf("Normalize(%q) = %q, want %q", in, got, want)
		}
	}
	// Add normalizes before storing, so a messy input matches a clean Pinned query.
	s := &Set{}
	s.Add("./agents/api/")
	if !s.Pinned("agents/api/agent.md") {
		t.Error("normalized pin should match nested path")
	}
}

func TestLoadMissingIsEmpty(t *testing.T) {
	s, err := Load(t.TempDir())
	if err != nil {
		t.Fatalf("Load missing: %v", err)
	}
	if len(s.Pins) != 0 {
		t.Errorf("missing file should load empty, got %d pins", len(s.Pins))
	}
	if s.SchemaVersion != SchemaVersion {
		t.Errorf("schemaVersion = %d, want %d", s.SchemaVersion, SchemaVersion)
	}
}

func TestWriteLoadRoundtrip(t *testing.T) {
	layer := t.TempDir()
	s := &Set{}
	s.Add("agents/api")
	s.Add("skills/build")
	if err := s.Write(layer); err != nil {
		t.Fatalf("Write: %v", err)
	}
	got, err := Load(layer)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(got.Pins) != 2 || !got.Pinned("agents/api/x.md") || !got.Pinned("skills/build") {
		t.Errorf("roundtrip mismatch: %+v", got.Pins)
	}
	// Written order is sorted for stability.
	if got.Pins[0] != "agents/api" || got.Pins[1] != "skills/build" {
		t.Errorf("pins not sorted: %+v", got.Pins)
	}
}
