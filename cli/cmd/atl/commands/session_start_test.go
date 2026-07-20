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

func TestBoardTrackedSignal(t *testing.T) {
	t.Run("fires with a backend named", func(t *testing.T) {
		root := t.TempDir()
		if err := os.MkdirAll(filepath.Join(root, ".delivery"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, ".delivery", "config.json"), []byte(`{"backend":"github"}`), 0o644); err != nil {
			t.Fatal(err)
		}
		out := captureStdout(t, func() { boardTrackedSignal(root) })
		if !strings.Contains(out, "board-backed (github)") || !strings.Contains(out, "board-tracked-work rule") {
			t.Fatalf("expected a github board-tracked signal, got %q", out)
		}
	})

	t.Run("defaults the backend to azure when unset", func(t *testing.T) {
		root := t.TempDir()
		_ = os.MkdirAll(filepath.Join(root, ".delivery"), 0o755)
		_ = os.WriteFile(filepath.Join(root, ".delivery", "config.json"), []byte(`{"org":"x"}`), 0o644)
		out := captureStdout(t, func() { boardTrackedSignal(root) })
		if !strings.Contains(out, "board-backed (azure)") {
			t.Fatalf("expected the azure default, got %q", out)
		}
	})

	t.Run("silent with no board backend", func(t *testing.T) {
		out := captureStdout(t, func() { boardTrackedSignal(t.TempDir()) })
		if out != "" {
			t.Fatalf("a project with no .delivery/config.json must be silent, got %q", out)
		}
	})

	t.Run("silent on empty root and malformed config", func(t *testing.T) {
		if out := captureStdout(t, func() { boardTrackedSignal("") }); out != "" {
			t.Fatalf("empty root must be silent, got %q", out)
		}
		root := t.TempDir()
		_ = os.MkdirAll(filepath.Join(root, ".delivery"), 0o755)
		_ = os.WriteFile(filepath.Join(root, ".delivery", "config.json"), []byte(`{not json`), 0o644)
		if out := captureStdout(t, func() { boardTrackedSignal(root) }); out != "" {
			t.Fatalf("a malformed config must be silent (never fail a hook), got %q", out)
		}
	})
}
