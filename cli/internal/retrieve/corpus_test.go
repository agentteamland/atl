package retrieve

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWalkCorpus(t *testing.T) {
	root := t.TempDir()
	write := func(rel, content string) {
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("wiki/merge-verify.md", "# Merge verify\n\nRead the durable branch state.")
	write("wiki/nested/dispatch.md", "no heading here, just body text")
	write("wiki/notes.txt", "not markdown, ignored")
	write("wiki/empty.md", "   \n  ")

	docs, err := WalkCorpus([]string{
		filepath.Join(root, "wiki"),
		filepath.Join(root, "does-not-exist"), // missing dir is skipped, not an error
	})
	if err != nil {
		t.Fatalf("WalkCorpus: %v", err)
	}
	if len(docs) != 2 {
		t.Fatalf("want 2 docs (md, non-empty), got %d: %+v", len(docs), docs)
	}
	// Sorted by absolute path: wiki/merge-verify.md < wiki/nested/dispatch.md.
	if docs[0].Title != "Merge verify" { // H1 heading
		t.Errorf("doc0 title: want 'Merge verify', got %q", docs[0].Title)
	}
	if docs[1].Title != "dispatch" { // no H1 -> filename
		t.Errorf("doc1 title: want filename 'dispatch', got %q", docs[1].Title)
	}
}

func TestWalkCorpusEmptyInput(t *testing.T) {
	docs, err := WalkCorpus(nil)
	if err != nil || len(docs) != 0 {
		t.Fatalf("nil dirs: want no docs no error, got %d docs, err %v", len(docs), err)
	}
}

func TestWalkCorpusSkipsVendorAndHidden(t *testing.T) {
	root := t.TempDir()
	write := func(rel string) {
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("# T\nbody"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("real.md")
	write("node_modules/dep/README.md") // vendored — must be skipped
	write(".cache/stale.md")            // hidden dir — must be skipped

	docs, err := WalkCorpus([]string{root})
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 1 || filepath.Base(docs[0].Path) != "real.md" {
		t.Fatalf("want only real.md, got %+v", docs)
	}
}
