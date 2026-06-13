package manifest

import (
	"path/filepath"
	"testing"
	"time"
)

func TestWriteRead(t *testing.T) {
	layer := t.TempDir()
	m := &Manifest{
		Handle:  "agentteamland",
		Name:    "software-project-team",
		Version: "1.2.1",
		Scope:   "project",
		Source:  Source{Repo: "agentteamland/software-project-team", Subpath: "", Ref: "v1.2.1"},
		Files:   map[string]string{"agents/api/agent.md": "abc123"},
	}
	if err := m.Write(layer); err != nil {
		t.Fatalf("Write: %v", err)
	}
	got, err := Read(layer, "agentteamland", "software-project-team")
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if got.Version != "1.2.1" || got.Files["agents/api/agent.md"] != "abc123" {
		t.Errorf("roundtrip mismatch: %+v", got)
	}
	if got.SchemaVersion != SchemaVersion {
		t.Errorf("schemaVersion = %d, want %d", got.SchemaVersion, SchemaVersion)
	}
	if got.InstalledAt.IsZero() {
		t.Error("installedAt not auto-set")
	}
}

func TestPath(t *testing.T) {
	p := Path(filepath.FromSlash("/root/.atl"), "mesut", "my-team")
	want := filepath.FromSlash("/root/.atl/installed/mesut__my-team.json")
	if p != want {
		t.Errorf("Path = %q, want %q", p, want)
	}
}

func TestList(t *testing.T) {
	layer := t.TempDir()
	mustWrite(t, layer, &Manifest{Handle: "b", Name: "two", Files: map[string]string{}})
	mustWrite(t, layer, &Manifest{Handle: "a", Name: "one", Files: map[string]string{}})
	ms, err := List(layer)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(ms) != 2 {
		t.Fatalf("List len = %d, want 2", len(ms))
	}
	if ms[0].Handle != "a" || ms[1].Handle != "b" {
		t.Errorf("List not sorted: %s, %s", ms[0].Handle, ms[1].Handle)
	}
}

func TestListEmpty(t *testing.T) {
	ms, err := List(t.TempDir())
	if err != nil {
		t.Fatalf("List empty: %v", err)
	}
	if len(ms) != 0 {
		t.Errorf("expected 0, got %d", len(ms))
	}
}

func TestRemove(t *testing.T) {
	layer := t.TempDir()
	mustWrite(t, layer, &Manifest{Handle: "a", Name: "one", Files: map[string]string{}})
	if err := Remove(layer, "a", "one"); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if _, err := Read(layer, "a", "one"); err == nil {
		t.Error("expected read-after-remove to fail")
	}
	if err := Remove(layer, "a", "one"); err != nil {
		t.Errorf("Remove should be idempotent, got %v", err)
	}
}

func TestInstalledAtPreserved(t *testing.T) {
	layer := t.TempDir()
	ts := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	mustWrite(t, layer, &Manifest{Handle: "a", Name: "one", InstalledAt: ts, Files: map[string]string{}})
	got, _ := Read(layer, "a", "one")
	if !got.InstalledAt.Equal(ts) {
		t.Errorf("installedAt = %v, want %v", got.InstalledAt, ts)
	}
}

func mustWrite(t *testing.T, layer string, m *Manifest) {
	t.Helper()
	if err := m.Write(layer); err != nil {
		t.Fatalf("Write: %v", err)
	}
}
