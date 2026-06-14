package coreassets

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

// TestReflect writes the embedded core into a temp dir and checks the expected
// rules + skills land at the right relative paths.
func TestReflect(t *testing.T) {
	dir := t.TempDir()
	n, err := Reflect(dir)
	if err != nil {
		t.Fatalf("Reflect: %v", err)
	}
	if n == 0 {
		t.Fatal("Reflect wrote 0 files")
	}
	for _, rel := range []string{
		"rules/communication-style.md",
		"rules/learning-capture.md",
		"skills/drain/SKILL.md",
		"skills/create-pr/SKILL.md",
	} {
		if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(rel))); err != nil {
			t.Errorf("expected %s reflected: %v", rel, err)
		}
	}
}

// TestEmbedMatchesCore guards against a stale mirror: every file under
// ../../../core/{rules,skills} must be present and identical in embed/. If this
// fails, run sync.sh.
func TestEmbedMatchesCore(t *testing.T) {
	for _, sub := range []string{"rules", "skills"} {
		coreDir := filepath.Join("..", "..", "..", "core", sub)
		walkErr := filepath.WalkDir(coreDir, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			rel, err := filepath.Rel(coreDir, p)
			if err != nil {
				return err
			}
			want, err := os.ReadFile(p)
			if err != nil {
				return err
			}
			embedPath := filepath.ToSlash(filepath.Join(embedRoot, sub, rel))
			got, err := assets.ReadFile(embedPath)
			if err != nil {
				t.Errorf("embed missing %s (run sync.sh): %v", embedPath, err)
				return nil
			}
			if string(got) != string(want) {
				t.Errorf("embed drifted from core for %s (run sync.sh)", embedPath)
			}
			return nil
		})
		if walkErr != nil {
			t.Fatalf("walk %s: %v", coreDir, walkErr)
		}
	}
}
