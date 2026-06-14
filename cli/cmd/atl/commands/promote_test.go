package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/agentteamland/atl/cli/internal/fanout"
	"github.com/agentteamland/atl/cli/internal/manifest"
	"github.com/agentteamland/atl/cli/internal/pin"
)

// TestPromoteGains proves the full apply path: a clean lift, a conflict lift
// (prior global archived, project wins), a new-file lift, a pinned skip, both
// manifests' baselines advancing, and a second pass being a no-op.
func TestPromoteGains(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	proj := t.TempDir()

	globClaude := filepath.Join(home, ".claude")
	projClaude := filepath.Join(proj, ".claude")
	globLayer := filepath.Join(home, ".atl")
	projLayer := filepath.Join(proj, ".atl")

	// Team installed at both scopes with a shared install baseline.
	baseline := map[string]string{
		"agents/a/agent.md":    fanout.Hash([]byte("V1")),
		"agents/a/keep.md":     fanout.Hash([]byte("KEEP")),
		"agents/a/conflict.md": fanout.Hash([]byte("C1")),
	}
	for _, layer := range []string{globLayer, projLayer} {
		scopeName := "global"
		if layer == projLayer {
			scopeName = "project"
		}
		m := &manifest.Manifest{Handle: "acme", Name: "demo", Scope: scopeName,
			Files: cloneMap(baseline)}
		if err := m.Write(layer); err != nil {
			t.Fatal(err)
		}
	}

	// Project evolved past baseline; global stayed put except conflict.md.
	writeF(t, filepath.Join(projClaude, "agents/a/agent.md"), "V2")         // modified → clean Lift
	writeF(t, filepath.Join(projClaude, "agents/a/keep.md"), "KEEP")        // unchanged → skip
	writeF(t, filepath.Join(projClaude, "agents/a/conflict.md"), "Cproj")   // modified → ConflictLift
	writeF(t, filepath.Join(projClaude, "agents/a/children/new.md"), "NEW") // new → Lift
	writeF(t, filepath.Join(projClaude, "agents/a/local.md"), "LOCAL")      // new but pinned → skip

	writeF(t, filepath.Join(globClaude, "agents/a/agent.md"), "V1")
	writeF(t, filepath.Join(globClaude, "agents/a/keep.md"), "KEEP")
	writeF(t, filepath.Join(globClaude, "agents/a/conflict.md"), "Cglob") // diverged from baseline

	// Pin the local-only file so it is never promoted.
	pins := &pin.Set{}
	pins.Add("agents/a/local.md")
	if err := pins.Write(projLayer); err != nil {
		t.Fatal(err)
	}

	res, err := promoteGains(proj, "")
	if err != nil {
		t.Fatalf("promoteGains: %v", err)
	}
	if res.lifted != 3 || res.conflicts != 1 || res.teams != 1 {
		t.Fatalf("result = %+v, want lifted=3 conflicts=1 teams=1", res)
	}

	// Global now holds the promoted project values.
	assertFile(t, filepath.Join(globClaude, "agents/a/agent.md"), "V2")
	assertFile(t, filepath.Join(globClaude, "agents/a/conflict.md"), "Cproj")
	assertFile(t, filepath.Join(globClaude, "agents/a/children/new.md"), "NEW")
	// Pinned file never reached global.
	if _, err := os.Stat(filepath.Join(globClaude, "agents/a/local.md")); !os.IsNotExist(err) {
		t.Error("pinned local.md should not have been promoted to global")
	}

	// Conflict: prior global value archived (content-addressed by its hash).
	archive := filepath.Join(globLayer, "history", "acme__demo", fanout.Hash([]byte("Cglob")), "agents/a/conflict.md")
	assertFile(t, archive, "Cglob")

	// Both manifests advanced their baselines to the promoted hashes, and the new
	// file is now tracked at both scopes.
	for _, layer := range []string{globLayer, projLayer} {
		m, err := manifest.Read(layer, "acme", "demo")
		if err != nil {
			t.Fatalf("read manifest in %s: %v", layer, err)
		}
		if m.Files["agents/a/agent.md"] != fanout.Hash([]byte("V2")) {
			t.Errorf("%s: agent.md baseline not advanced", layer)
		}
		if m.Files["agents/a/conflict.md"] != fanout.Hash([]byte("Cproj")) {
			t.Errorf("%s: conflict.md baseline not advanced", layer)
		}
		if m.Files["agents/a/children/new.md"] != fanout.Hash([]byte("NEW")) {
			t.Errorf("%s: new.md not tracked", layer)
		}
		if _, ok := m.Files["agents/a/local.md"]; ok {
			t.Errorf("%s: pinned local.md should not be tracked", layer)
		}
	}

	// Idempotency: nothing left to lift on a second pass.
	res2, err := promoteGains(proj, "")
	if err != nil {
		t.Fatalf("second promoteGains: %v", err)
	}
	if res2.lifted != 0 {
		t.Errorf("second pass lifted %d, want 0 (idempotent)", res2.lifted)
	}
}

// TestPromoteGainsProjectOnly: a team with no global install has no upstream to
// lift into and is left alone.
func TestPromoteGainsProjectOnly(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	proj := t.TempDir()

	projLayer := filepath.Join(proj, ".atl")
	projClaude := filepath.Join(proj, ".claude")
	m := &manifest.Manifest{Handle: "acme", Name: "solo", Scope: "project",
		Files: map[string]string{"agents/a/agent.md": fanout.Hash([]byte("V1"))}}
	if err := m.Write(projLayer); err != nil {
		t.Fatal(err)
	}
	writeF(t, filepath.Join(projClaude, "agents/a/agent.md"), "V2") // evolved, but no global copy

	res, err := promoteGains(proj, "")
	if err != nil {
		t.Fatalf("promoteGains: %v", err)
	}
	if res.lifted != 0 {
		t.Errorf("project-only team should not promote, lifted %d", res.lifted)
	}
}

func cloneMap(m map[string]string) map[string]string {
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func assertFile(t *testing.T, path, want string) {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if string(b) != want {
		t.Errorf("%s = %q, want %q", path, b, want)
	}
}
