package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/agentteamland/atl/cli/internal/fanout"
)

func TestReflectWithFanout(t *testing.T) {
	src := t.TempDir()
	claude := t.TempDir()
	w := func(dir, rel, body string) {
		p := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// upstream = the new version
	w(src, "agents/a/unchanged.md", "NEW-A")
	w(src, "agents/b/edited.md", "NEW-B")
	w(src, "agents/c/brandnew.md", "C")
	w(src, "knowledge/adapter.md", "NEW-K") // non-agent/skill/rule asset must refresh too
	// installed copy
	w(claude, "agents/a/unchanged.md", "OLD-A")  // == baseline → refresh
	w(claude, "agents/b/edited.md", "USER-EDIT") // != baseline → preserve
	w(claude, "knowledge/adapter.md", "OLD-K")   // == baseline → refresh
	baseline := map[string]string{
		"agents/a/unchanged.md": fanout.Hash([]byte("OLD-A")),
		"agents/b/edited.md":    fanout.Hash([]byte("OLD-B")),
		"knowledge/adapter.md":  fanout.Hash([]byte("OLD-K")),
	}

	next, err := reflectWithFanout(src, claude, baseline)
	if err != nil {
		t.Fatalf("reflectWithFanout: %v", err)
	}
	if b, _ := os.ReadFile(filepath.Join(claude, "agents/a/unchanged.md")); string(b) != "NEW-A" {
		t.Errorf("unchanged should refresh to NEW-A, got %q", b)
	}
	if b, _ := os.ReadFile(filepath.Join(claude, "agents/b/edited.md")); string(b) != "USER-EDIT" {
		t.Errorf("edited should be preserved, got %q", b)
	}
	if b, _ := os.ReadFile(filepath.Join(claude, "agents/c/brandnew.md")); string(b) != "C" {
		t.Errorf("brandnew should be added, got %q", b)
	}
	if b, _ := os.ReadFile(filepath.Join(claude, "knowledge/adapter.md")); string(b) != "NEW-K" {
		t.Errorf("knowledge asset should refresh to NEW-K, got %q", b)
	}
	if next["knowledge/adapter.md"] != fanout.Hash([]byte("NEW-K")) {
		t.Error("knowledge asset must stay in the manifest (advance to NEW-K), not be dropped")
	}
	if next["agents/a/unchanged.md"] != fanout.Hash([]byte("NEW-A")) {
		t.Error("baseline a should advance to NEW-A")
	}
	if next["agents/b/edited.md"] != fanout.Hash([]byte("OLD-B")) {
		t.Error("baseline b should stay at the original install point (preserve)")
	}
	if next["agents/c/brandnew.md"] != fanout.Hash([]byte("C")) {
		t.Error("baseline c should be the new file's hash")
	}
}

// TestReflectWithFanoutPreservesLocalCollision: a brand-new upstream file at a
// path the install never owned, but where the user (or the learning loop) already
// created a divergent local file, must NOT be clobbered — copying over it would
// silently destroy local work — and the preserved file must not be claimed in the
// manifest. Regression for the data-loss the old unconditional-copy `!known`
// branch caused.
func TestReflectWithFanoutPreservesLocalCollision(t *testing.T) {
	src := t.TempDir()
	claude := t.TempDir()
	w := func(dir, rel, body string) {
		p := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// The new version now ships knowledge/notes.md — but the user already has a
	// different local file at that path, and it was never in the install baseline.
	w(src, "knowledge/notes.md", "UPSTREAM")
	w(claude, "knowledge/notes.md", "USER")
	// A genuinely-new file with no local copy must still be added.
	w(src, "knowledge/fresh.md", "FRESH")
	baseline := map[string]string{}

	next, err := reflectWithFanout(src, claude, baseline)
	if err != nil {
		t.Fatalf("reflectWithFanout: %v", err)
	}
	if b, _ := os.ReadFile(filepath.Join(claude, "knowledge/notes.md")); string(b) != "USER" {
		t.Fatalf("a colliding local file must be preserved, not clobbered; got %q", b)
	}
	if _, owned := next["knowledge/notes.md"]; owned {
		t.Error("a preserved local file must not be recorded in the manifest")
	}
	if b, _ := os.ReadFile(filepath.Join(claude, "knowledge/fresh.md")); string(b) != "FRESH" {
		t.Errorf("a genuinely-new file (no local) must still be added, got %q", b)
	}
	if next["knowledge/fresh.md"] != fanout.Hash([]byte("FRESH")) {
		t.Error("the new file should be recorded in the manifest")
	}
}
