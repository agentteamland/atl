package integrity

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/agentteamland/atl/cli/internal/fanout"
	"github.com/agentteamland/atl/cli/internal/manifest"
)

func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestScanFindsAbsent(t *testing.T) {
	layer := t.TempDir()
	claude := t.TempDir()
	writeFile(t, filepath.Join(claude, "agents/a/present.md"), "P")
	m := &manifest.Manifest{Handle: "acme", Name: "demo",
		Source: manifest.Source{Repo: "acme/demo", Ref: "v1"},
		Files: map[string]string{
			"agents/a/present.md": fanout.Hash([]byte("P")),
			"agents/a/gone.md":    "deadbeef",
		}}
	if err := m.Write(layer); err != nil {
		t.Fatal(err)
	}
	missing, err := Scan(layer, claude)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(missing) != 1 || missing[0].Rel != "agents/a/gone.md" {
		t.Fatalf("missing = %+v, want only gone.md", missing)
	}
	if missing[0].Source.Repo != "acme/demo" {
		t.Errorf("missing.Source = %+v", missing[0].Source)
	}
}

func TestScanIgnoresModified(t *testing.T) {
	// present-but-changed is a user edit, not drift — must NOT be reported.
	layer := t.TempDir()
	claude := t.TempDir()
	writeFile(t, filepath.Join(claude, "x.md"), "USER-CHANGED")
	m := &manifest.Manifest{Handle: "a", Name: "b",
		Files: map[string]string{"x.md": fanout.Hash([]byte("ORIGINAL"))}}
	if err := m.Write(layer); err != nil {
		t.Fatal(err)
	}
	missing, err := Scan(layer, claude)
	if err != nil {
		t.Fatal(err)
	}
	if len(missing) != 0 {
		t.Errorf("modified-but-present should not be drift, got %+v", missing)
	}
}

func TestRestoreFromDir(t *testing.T) {
	src := t.TempDir()
	claude := t.TempDir()
	writeFile(t, filepath.Join(src, "agents/a/gone.md"), "RESTORED")
	m := Missing{Rel: "agents/a/gone.md", SHA: fanout.Hash([]byte("RESTORED"))}
	if err := restoreFromDir(src, m, claude); err != nil {
		t.Fatalf("restoreFromDir: %v", err)
	}
	b, _ := os.ReadFile(filepath.Join(claude, "agents/a/gone.md"))
	if string(b) != "RESTORED" {
		t.Errorf("restored content = %q", b)
	}
}

func TestRestoreChecksumMismatch(t *testing.T) {
	src := t.TempDir()
	writeFile(t, filepath.Join(src, "x.md"), "WRONG-CONTENT")
	m := Missing{Rel: "x.md", SHA: fanout.Hash([]byte("EXPECTED"))}
	if err := restoreFromDir(src, m, t.TempDir()); err == nil {
		t.Error("expected checksum-mismatch error")
	}
}
