package userrules

import (
	"os"
	"path/filepath"
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

func TestReflectCopiesMarkdownOnly(t *testing.T) {
	layer := t.TempDir()
	claude := t.TempDir()
	write(t, filepath.Join(layer, "rules", "house-style.md"), "rule body")
	write(t, filepath.Join(layer, "rules", "notes.txt"), "not a rule")     // non-md → skipped
	write(t, filepath.Join(layer, "rules", "sub", "nested.md"), "in a dir") // dir entry → skipped

	n, err := Reflect(layer, claude, nil)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("want 1 file reflected, got %d", n)
	}
	got, err := os.ReadFile(filepath.Join(claude, "rules", "house-style.md"))
	if err != nil || string(got) != "rule body" {
		t.Fatalf("house-style.md not reflected correctly: %q err=%v", got, err)
	}
	if _, err := os.Stat(filepath.Join(claude, "rules", "notes.txt")); !os.IsNotExist(err) {
		t.Error("a non-.md file must not be reflected")
	}
	if _, err := os.Stat(filepath.Join(claude, "rules", "sub")); !os.IsNotExist(err) {
		t.Error("a subdirectory must not be recursed into")
	}
}

func TestReflectIdempotentSkipsUnchanged(t *testing.T) {
	layer := t.TempDir()
	claude := t.TempDir()
	write(t, filepath.Join(layer, "rules", "a.md"), "same")

	if n, err := Reflect(layer, claude, nil); err != nil || n != 1 {
		t.Fatalf("first reflect: want 1, got %d err=%v", n, err)
	}
	// Second reflect with unchanged content must write nothing.
	if n, err := Reflect(layer, claude, nil); err != nil || n != 0 {
		t.Fatalf("second reflect must skip unchanged: want 0, got %d err=%v", n, err)
	}
	// Change the source → it reflects again.
	write(t, filepath.Join(layer, "rules", "a.md"), "changed")
	if n, err := Reflect(layer, claude, nil); err != nil || n != 1 {
		t.Fatalf("changed source must re-reflect: want 1, got %d err=%v", n, err)
	}
}

func TestReflectProtectedNamesSkipped(t *testing.T) {
	layer := t.TempDir()
	claude := t.TempDir()
	write(t, filepath.Join(layer, "rules", "branch-hygiene.md"), "user override")
	write(t, filepath.Join(layer, "rules", "mine.md"), "my rule")

	n, err := Reflect(layer, claude, map[string]bool{"branch-hygiene.md": true})
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("want 1 reflected (protected one skipped), got %d", n)
	}
	if _, err := os.Stat(filepath.Join(claude, "rules", "branch-hygiene.md")); !os.IsNotExist(err) {
		t.Error("a protected (core-named) rule must not be reflected over core")
	}
	if _, err := os.Stat(filepath.Join(claude, "rules", "mine.md")); err != nil {
		t.Error("a non-protected user rule must still be reflected")
	}
}

func TestReflectMissingDirIsNoError(t *testing.T) {
	layer := t.TempDir() // no rules/ subdir
	claude := t.TempDir()
	n, err := Reflect(layer, claude, nil)
	if err != nil || n != 0 {
		t.Fatalf("missing source dir must be a no-op: got %d err=%v", n, err)
	}
}

func TestOwned(t *testing.T) {
	layer := t.TempDir()
	write(t, filepath.Join(layer, "rules", "one.md"), "x")
	write(t, filepath.Join(layer, "rules", "two.md"), "y")
	write(t, filepath.Join(layer, "rules", "skip.txt"), "z")

	owned, err := Owned(layer)
	if err != nil {
		t.Fatal(err)
	}
	if !owned["rules/one.md"] || !owned["rules/two.md"] {
		t.Errorf("both .md rules should be owned: %v", owned)
	}
	if owned["rules/skip.txt"] {
		t.Error("a non-.md file must not be reported as an owned rule")
	}
	if len(owned) != 2 {
		t.Errorf("want exactly 2 owned rules, got %d: %v", len(owned), owned)
	}

	// A layer with no rules/ dir yields an empty set, not an error.
	empty, err := Owned(t.TempDir())
	if err != nil || len(empty) != 0 {
		t.Fatalf("missing dir should yield empty set: %v err=%v", empty, err)
	}
}
