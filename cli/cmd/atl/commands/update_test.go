package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/agentteamland/atl/cli/internal/fanout"
	"github.com/agentteamland/atl/cli/internal/manifest"
	"github.com/agentteamland/atl/cli/internal/pin"
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

// TestFanOutUnionAddsGlobalOnlyGain proves the union half: a file the global
// layer gained (a promoted gain the project doesn't have yet) fans out into the
// project even though it isn't in the project baseline — closing promote's ring 2->3.
func TestFanOutUnionAddsGlobalOnlyGain(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	proj := t.TempDir()

	globalClaude := filepath.Join(home, ".claude")
	projClaude := filepath.Join(proj, ".claude")
	globalLayer := filepath.Join(home, ".atl")
	projLayer := filepath.Join(proj, ".atl")

	// global holds a promoted gain the project's manifest doesn't list.
	writeF(t, filepath.Join(globalClaude, "agents/a/gain.md"), "PROMOTED-GAIN")
	gainHash := fanout.Hash([]byte("PROMOTED-GAIN"))
	gm := &manifest.Manifest{Handle: "acme", Name: "demo", Scope: "global",
		Files: map[string]string{"agents/a/gain.md": gainHash}}
	if err := gm.Write(globalLayer); err != nil {
		t.Fatal(err)
	}
	pm := &manifest.Manifest{Handle: "acme", Name: "demo", Scope: "project",
		Files: map[string]string{"agents/a/base.md": "somehash"}} // team present, gain absent
	if err := pm.Write(projLayer); err != nil {
		t.Fatal(err)
	}

	n, err := fanOut(proj)
	if err != nil {
		t.Fatalf("fanOut: %v", err)
	}
	if n != 1 {
		t.Errorf("refreshed = %d, want 1 (the global-only gain)", n)
	}
	if b, _ := os.ReadFile(filepath.Join(projClaude, "agents/a/gain.md")); string(b) != "PROMOTED-GAIN" {
		t.Errorf("global-only gain not fanned out: %q", b)
	}
	got, _ := manifest.Read(projLayer, "acme", "demo")
	if got.Files["agents/a/gain.md"] != gainHash {
		t.Error("project manifest should record the fanned-out gain baseline")
	}
}

// TestFanOutUnionRespectsPin: a project that pinned a path keeps it project-free —
// a global-only gain at a pinned path is not fanned in.
func TestFanOutUnionRespectsPin(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	proj := t.TempDir()

	globalClaude := filepath.Join(home, ".claude")
	projClaude := filepath.Join(proj, ".claude")
	globalLayer := filepath.Join(home, ".atl")
	projLayer := filepath.Join(proj, ".atl")

	writeF(t, filepath.Join(globalClaude, "agents/a/gain.md"), "PROMOTED-GAIN")
	gm := &manifest.Manifest{Handle: "acme", Name: "demo", Scope: "global",
		Files: map[string]string{"agents/a/gain.md": fanout.Hash([]byte("PROMOTED-GAIN"))}}
	if err := gm.Write(globalLayer); err != nil {
		t.Fatal(err)
	}
	pm := &manifest.Manifest{Handle: "acme", Name: "demo", Scope: "project",
		Files: map[string]string{"agents/a/base.md": "somehash"}}
	if err := pm.Write(projLayer); err != nil {
		t.Fatal(err)
	}
	pins := &pin.Set{}
	pins.Add("agents/a/gain.md")
	if err := pins.Write(projLayer); err != nil {
		t.Fatal(err)
	}

	n, err := fanOut(proj)
	if err != nil {
		t.Fatalf("fanOut: %v", err)
	}
	if n != 0 {
		t.Errorf("a pinned global-only gain must not fan out, got %d", n)
	}
	if _, err := os.Stat(filepath.Join(projClaude, "agents/a/gain.md")); !os.IsNotExist(err) {
		t.Error("pinned gain should not be copied into the project")
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
