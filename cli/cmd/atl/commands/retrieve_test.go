package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/agentteamland/atl/cli/internal/retrieve"
)

func TestFormatResults(t *testing.T) {
	root := t.TempDir()
	p := filepath.Join(root, ".atl", "wiki", "x.md")
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("# Title Here\n\nBody line one is the excerpt."), 0o644); err != nil {
		t.Fatal(err)
	}
	out := formatResults(root, []retrieve.Result{{Path: p, Title: "Title Here"}})
	for _, want := range []string{"atl#140", "Title Here", ".atl/wiki/x.md", "Body line one"} {
		if !strings.Contains(out, want) {
			t.Errorf("formatResults missing %q in:\n%s", want, out)
		}
	}
}

func TestExcerptSkipsFrontmatterAndHeadings(t *testing.T) {
	p := filepath.Join(t.TempDir(), "y.md")
	if err := os.WriteFile(p, []byte("---\nknowledge-base-summary: meta text\n---\n# Heading\n\nReal prose body."), 0o644); err != nil {
		t.Fatal(err)
	}
	ex := excerpt(p)
	if strings.Contains(ex, "knowledge-base-summary") {
		t.Errorf("frontmatter leaked into excerpt: %q", ex)
	}
	if strings.Contains(ex, "Heading") {
		t.Errorf("heading leaked into excerpt: %q", ex)
	}
	if !strings.Contains(ex, "Real prose body") {
		t.Errorf("excerpt missing body: %q", ex)
	}
}

func TestIndexPathFor(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	p, err := indexPathFor("/some/project")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(p, "index.gob") || !strings.Contains(filepath.ToSlash(p), "cache/retrieve") {
		t.Fatalf("unexpected index path: %s", p)
	}
}
