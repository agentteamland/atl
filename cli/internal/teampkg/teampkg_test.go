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
	writeFile(t, filepath.Join(src, "knowledge/adapter.md"), "ADAPTER")
	writeFile(t, filepath.Join(src, "scripts/helper.sh"), "#!/usr/bin/env bash\necho hi\n")
	writeFile(t, filepath.Join(src, "packs/web/pack.md"), "PACK")                // M1 knowledge-pack seam — must reflect too
	writeFile(t, filepath.Join(src, "backends/github/adapter.md"), "GH-ADAPTER") // per-backend adapter contract — must reflect too

	claude := t.TempDir()
	files, err := CopyAssets(src, claude)
	if err != nil {
		t.Fatalf("CopyAssets: %v", err)
	}
	for _, rel := range []string{"agents/api/agent.md", "skills/build/skill.md", "rules/r.md", "knowledge/adapter.md", "scripts/helper.sh", "packs/web/pack.md", "backends/github/adapter.md"} {
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

func TestCopyFilePreservesExecBit(t *testing.T) {
	src := filepath.Join(t.TempDir(), "helper.sh")
	if err := os.WriteFile(src, []byte("#!/usr/bin/env bash\necho hi\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(t.TempDir(), "helper.sh")
	if err := CopyFile(src, dst); err != nil {
		t.Fatalf("CopyFile: %v", err)
	}
	fi, err := os.Stat(dst)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm() != 0o755 {
		t.Errorf("exec bit not preserved: dst mode = %v, want 0755", fi.Mode().Perm())
	}
}

func TestReadManifestDependencies(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "team.json"),
		`{"name":"consumer","version":"1.0.0","dependencies":{"core":"^1.0.0","profile-team":"^1.1.0"}}`)
	tm, err := ReadManifest(dir)
	if err != nil {
		t.Fatalf("ReadManifest: %v", err)
	}
	if tm.Dependencies["profile-team"] != "^1.1.0" || tm.Dependencies["core"] != "^1.0.0" {
		t.Errorf("dependencies = %+v", tm.Dependencies)
	}
}
