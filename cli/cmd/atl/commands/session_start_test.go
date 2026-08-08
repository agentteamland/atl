package commands

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// captureStdout runs fn with os.Stdout redirected to a pipe and returns what it wrote.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	fn()
	w.Close()
	os.Stdout = orig
	out, _ := io.ReadAll(r)
	return string(out)
}


func TestInstalledChildrenSignal(t *testing.T) {
	// Isolate the global layer so only the project layer under test contributes.
	writeChild := func(t *testing.T, claude, agent, name, body string) {
		t.Helper()
		dir := filepath.Join(claude, "agents", agent)
		if err := os.MkdirAll(filepath.Join(dir, "children"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "agent.md"), []byte("---\nname: "+agent+"\ndescription: \"x\"\n---\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "children", name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	t.Run("fires on a child with no summary", func(t *testing.T) {
		t.Setenv("HOME", t.TempDir())
		root := t.TempDir()
		writeChild(t, filepath.Join(root, ".claude"), "api", "bad.md", "# no frontmatter\n")

		out := captureStdout(t, func() { installedChildrenSignal(root) })
		if !strings.Contains(out, "1 agent-KB child(ren)") || !strings.Contains(out, "knowledge-base-summary") {
			t.Fatalf("expected an installed-children signal, got %q", out)
		}
	})

	t.Run("silent when every child declares its summary", func(t *testing.T) {
		t.Setenv("HOME", t.TempDir())
		root := t.TempDir()
		writeChild(t, filepath.Join(root, ".claude"), "api", "good.md", "---\nknowledge-base-summary: \"a summary\"\n---\n")

		if out := captureStdout(t, func() { installedChildrenSignal(root) }); out != "" {
			t.Fatalf("a clean installed layer must be silent, got %q", out)
		}
	})

	t.Run("silent with no project root", func(t *testing.T) {
		t.Setenv("HOME", t.TempDir())
		if out := captureStdout(t, func() { installedChildrenSignal("") }); out != "" {
			t.Fatalf("no project root must be silent, got %q", out)
		}
	})

	// The project root is the session's cwd, so a session started in $HOME makes
	// both layers the same directory — one broken child must count once.
	t.Run("counts a shared layer once", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("HOME", home)
		writeChild(t, filepath.Join(home, ".claude"), "api", "bad.md", "# no frontmatter\n")

		out := captureStdout(t, func() { installedChildrenSignal(home) })
		if !strings.Contains(out, "1 agent-KB child(ren)") {
			t.Fatalf("$HOME as project root must not double-count, got %q", out)
		}
	})
}
