package rulesscan

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestCollectFindsNormativeAndSkipsRules(t *testing.T) {
	root := t.TempDir()
	core := filepath.Join(root, "core")
	// a skill with one normative line + one plain line + a heading
	write(t, filepath.Join(core, "skills/demo/SKILL.md"), "# Demo\nAlways grep before you edit.\nThis is a plain sentence.\n")
	// a rule — the distill TARGET, must be skipped even though it's normative
	write(t, filepath.Join(core, "rules/some-rule.md"), "You must never do this.\n")

	stmts, err := Collect(core, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(stmts) != 1 {
		t.Fatalf("want exactly 1 normative statement (rules/ skipped, heading + plain ignored), got %d: %+v", len(stmts), stmts)
	}
	if stmts[0].File != "core/skills/demo/SKILL.md" || !strings.Contains(stmts[0].Text, "Always grep") {
		t.Errorf("wrong statement: %+v", stmts[0])
	}
}

func TestCollectMissingDirIsEmpty(t *testing.T) {
	stmts, err := Collect(filepath.Join(t.TempDir(), "nope"), "")
	if err != nil {
		t.Fatalf("missing dir should not error: %v", err)
	}
	if len(stmts) != 0 {
		t.Fatalf("want none, got %+v", stmts)
	}
}
