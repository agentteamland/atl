package retrieve

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestIndexSaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "index.gob") // Save creates parent dirs

	orig := &Index{
		Version: indexFormatVersion,
		Docs: []Doc{
			{Path: "a.md", Title: "A", Text: "alpha body"},
			{Path: "b.md", Title: "B", Text: "beta body"},
		},
		Vecs: [][]float32{{0.1, 0.2}, {0.3, 0.4}},
	}
	if err := orig.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got == nil || len(got.Docs) != 2 || got.Docs[1].Title != "B" || got.Vecs[0][1] != 0.2 {
		t.Fatalf("round-trip mismatch: %+v", got)
	}
}

func TestLoadMissingAndCorruptAreAbsent(t *testing.T) {
	dir := t.TempDir()

	// Missing file -> (nil, nil), the "rebuild me" signal.
	got, err := Load(filepath.Join(dir, "nope.gob"))
	if got != nil || err != nil {
		t.Fatalf("missing: want (nil,nil), got (%v,%v)", got, err)
	}

	// Corrupt file -> (nil, nil), not an error (the index is a cache).
	bad := filepath.Join(dir, "bad.gob")
	if err := os.WriteFile(bad, []byte("not gob at all"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err = Load(bad)
	if got != nil || err != nil {
		t.Fatalf("corrupt: want (nil,nil), got (%v,%v)", got, err)
	}

	// Wrong version -> (nil, nil).
	stale := filepath.Join(dir, "stale.gob")
	if err := (&Index{Version: indexFormatVersion + 1}).Save(stale); err != nil {
		t.Fatal(err)
	}
	got, err = Load(stale)
	if got != nil || err != nil {
		t.Fatalf("stale version: want (nil,nil), got (%v,%v)", got, err)
	}
}

func TestQueryEmptyIndex(t *testing.T) {
	ix := &Index{Version: indexFormatVersion}
	got, err := ix.Query(context.Background(), "anything", nil, 5)
	if err != nil || len(got) != 0 {
		t.Fatalf("empty index query: want no results no error, got %d, %v", len(got), err)
	}
}

// TestBuildQueryEndToEnd exercises the full hybrid path with the real embedder.
// It skips when the model is not downloaded locally (CI stays offline).
func TestBuildQueryEndToEnd(t *testing.T) {
	dir := modelDirIfPresent(t)
	ctx := context.Background()
	e, err := NewEmbedder(ctx, dir)
	if err != nil {
		t.Fatalf("NewEmbedder: %v", err)
	}
	defer e.Close()

	docs := []Doc{
		{Path: "merge.md", Title: "Merge verify", Text: "The deterministic supervisor confirms a git merge by reading the durable branch state, never trusting an LLM worker's exit code."},
		{Path: "profile.md", Title: "Profile tiers", Text: "Profile facts are gated by privacy tier; the emotional state field is written only from a user-confirmed source."},
		{Path: "bread.md", Title: "Banana bread", Text: "The best banana bread recipe uses very ripe bananas and a pinch of cinnamon."},
	}
	ix, err := Build(ctx, docs, e)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if len(ix.Vecs) != 3 || len(ix.Vecs[0]) != 384 {
		t.Fatalf("bad vectors: %d docs, dim %d", len(ix.Vecs), len(ix.Vecs[0]))
	}

	// A conceptual query (no shared keyword with the doc's exact wording beyond
	// "merge") must still surface the merge-verify page first — the hybrid win.
	res, err := ix.Query(ctx, "how does the dispatch confirm a branch was merged", e, 2)
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(res) == 0 || res[0].Path != "merge.md" {
		t.Fatalf("expected merge.md first, got %+v", res)
	}
}
