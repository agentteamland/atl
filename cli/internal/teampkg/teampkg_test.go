package teampkg

import (
	"os"
	"path/filepath"
	"testing"
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

func TestReadManifest(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "team.json"),
		`{"schemaVersion":1,"name":"x","version":"1.0.0","scope":"global","capabilities":{"review":"r"}}`)
	tm, err := ReadManifest(dir)
	if err != nil {
		t.Fatalf("ReadManifest: %v", err)
	}
	if tm.Name != "x" || tm.Scope != "global" || tm.Version != "1.0.0" {
		t.Errorf("got %+v", tm)
	}
}

func TestReadManifestMissingName(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "team.json"), `{"version":"1.0.0"}`)
	if _, err := ReadManifest(dir); err == nil {
		t.Error("expected error for missing name")
	}
}

func TestCopyAssets(t *testing.T) {
	src := t.TempDir()
	writeFile(t, filepath.Join(src, "team.json"), `{"name":"x"}`)
	writeFile(t, filepath.Join(src, "README.md"), "readme")
	writeFile(t, filepath.Join(src, "agents/api/agent.md"), "API")
	writeFile(t, filepath.Join(src, "skills/build/skill.md"), "BUILD")
	writeFile(t, filepath.Join(src, "rules/r.md"), "RULE")

	claude := t.TempDir()
	files, err := CopyAssets(src, claude)
	if err != nil {
		t.Fatalf("CopyAssets: %v", err)
	}
	for _, rel := range []string{"agents/api/agent.md", "skills/build/skill.md", "rules/r.md"} {
		if _, err := os.Stat(filepath.Join(claude, rel)); err != nil {
			t.Errorf("missing copied %s: %v", rel, err)
		}
		if files[rel] == "" {
			t.Errorf("files map missing %s", rel)
		}
	}
	if _, err := os.Stat(filepath.Join(claude, "team.json")); err == nil {
		t.Error("team.json should not be copied")
	}
	if _, err := os.Stat(filepath.Join(claude, "README.md")); err == nil {
		t.Error("README.md should not be copied")
	}
	if files["team.json"] != "" {
		t.Error("team.json should not be in files map")
	}
}

func TestCopyAssetsNoAssets(t *testing.T) {
	src := t.TempDir()
	writeFile(t, filepath.Join(src, "team.json"), `{"name":"x"}`)
	if _, err := CopyAssets(src, t.TempDir()); err == nil {
		t.Error("expected error when team ships no assets")
	}
}
