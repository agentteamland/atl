package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/agentteamland/atl/cli/internal/fanout"
	"github.com/agentteamland/atl/cli/internal/manifest"
)

func writeF(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestFanOut proves the core rule: an unmodified project file refreshes from the
// global layer, a user-modified one is preserved.
func TestFanOut(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	proj := t.TempDir()

	globalClaude := filepath.Join(home, ".claude")
	projClaude := filepath.Join(proj, ".claude")
	globalLayer := filepath.Join(home, ".atl")
	projLayer := filepath.Join(proj, ".atl")

	// global has newer content for unchanged.md; baseline content for modded.md
	writeF(t, filepath.Join(globalClaude, "agents/a/unchanged.md"), "NEW-GLOBAL")
	writeF(t, filepath.Join(globalClaude, "agents/a/modded.md"), "ORIG")
	// project: unchanged.md still at baseline (eligible to refresh);
	//          modded.md changed by the user (must be preserved)
	writeF(t, filepath.Join(projClaude, "agents/a/unchanged.md"), "ORIG")
	writeF(t, filepath.Join(projClaude, "agents/a/modded.md"), "USER-EDIT")

	origHash := fanout.Hash([]byte("ORIG"))
	newHash := fanout.Hash([]byte("NEW-GLOBAL"))
	gm := &manifest.Manifest{Handle: "acme", Name: "demo", Scope: "global",
		Files: map[string]string{"agents/a/unchanged.md": newHash, "agents/a/modded.md": origHash}}
	if err := gm.Write(globalLayer); err != nil {
		t.Fatal(err)
	}
	pm := &manifest.Manifest{Handle: "acme", Name: "demo", Scope: "project",
		Files: map[string]string{"agents/a/unchanged.md": origHash, "agents/a/modded.md": origHash}}
	if err := pm.Write(projLayer); err != nil {
		t.Fatal(err)
	}

	n, err := fanOut(proj)
	if err != nil {
		t.Fatalf("fanOut: %v", err)
	}
	if n != 1 {
		t.Errorf("refreshed = %d, want 1", n)
	}
	if b, _ := os.ReadFile(filepath.Join(projClaude, "agents/a/unchanged.md")); string(b) != "NEW-GLOBAL" {
		t.Errorf("unchanged.md = %q, want NEW-GLOBAL (refreshed)", b)
	}
	if b, _ := os.ReadFile(filepath.Join(projClaude, "agents/a/modded.md")); string(b) != "USER-EDIT" {
		t.Errorf("modded.md = %q, want USER-EDIT (preserved)", b)
	}
	got, _ := manifest.Read(projLayer, "acme", "demo")
	if got.Files["agents/a/unchanged.md"] != newHash {
		t.Error("baseline not advanced after refresh")
	}
}

// TestFanOutNoGlobal: a project-only team has no upstream and is left alone.
func TestFanOutNoGlobal(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	proj := t.TempDir()
	pm := &manifest.Manifest{Handle: "acme", Name: "solo", Scope: "project",
		Files: map[string]string{"agents/a/x.md": "abc"}}
	if err := pm.Write(filepath.Join(proj, ".atl")); err != nil {
		t.Fatal(err)
	}
	n, err := fanOut(proj)
	if err != nil {
		t.Fatalf("fanOut: %v", err)
	}
	if n != 0 {
		t.Errorf("project-only team should not fan out, got %d", n)
	}
}
