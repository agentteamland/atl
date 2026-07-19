package retrieve

import (
	"context"
	"math"
	"path/filepath"
	"testing"
)

// modelDirIfPresent returns the default model directory when it is already
// downloaded and verified locally, or skips the test. Tests never hit the
// network: CI has no model and must stay offline and deterministic. Exercise
// the embedder locally by running `atl retrieve warm` first.
func modelDirIfPresent(t *testing.T) string {
	t.Helper()
	root, err := modelsRoot()
	if err != nil {
		t.Skipf("no models root: %v", err)
	}
	dir := filepath.Join(root, miniLMInt8.dir)
	for _, f := range miniLMInt8.files {
		if verifyFile(filepath.Join(dir, f.name), f.sha256) != nil {
			t.Skip("embedding model not present locally; run `atl retrieve warm` first")
		}
	}
	return dir
}

func TestEmbedDeterministicAndRanks(t *testing.T) {
	dir := modelDirIfPresent(t)
	ctx := context.Background()

	e, err := NewEmbedder(ctx, dir)
	if err != nil {
		t.Fatalf("NewEmbedder: %v", err)
	}
	defer e.Close()

	query := "how does the dispatch merge-verify work"
	relevant := "The deterministic supervisor confirms a git merge by reading the durable branch state, never trusting an LLM worker's exit code."
	irrelevant := "The best banana bread recipe uses very ripe bananas and a pinch of cinnamon."

	vecs, err := e.Embed(ctx, []string{query, relevant, irrelevant})
	if err != nil {
		t.Fatalf("embed: %v", err)
	}
	if len(vecs) != 3 {
		t.Fatalf("want 3 vectors, got %d", len(vecs))
	}
	if len(vecs[0]) != 384 {
		t.Fatalf("want dim 384 for MiniLM-L6-v2, got %d", len(vecs[0]))
	}

	// Determinism: re-embedding the SAME batch is bit-exact, so a vector index
	// built the same way is reproducible and cacheable. Embeddings are batch-order
	// stable but NOT invariant to batch composition — the same text in a batch
	// padded to a longer max-sequence-length can differ by float noise (which never
	// changes ranking), so the guarantee to assert is same-batch, not cross-batch.
	again, err := e.Embed(ctx, []string{query, relevant, irrelevant})
	if err != nil {
		t.Fatalf("embed again: %v", err)
	}
	for r := range vecs {
		for i := range vecs[r] {
			if vecs[r][i] != again[r][i] {
				t.Fatalf("non-deterministic embedding (row %d index %d): %v != %v", r, i, vecs[r][i], again[r][i])
			}
		}
	}

	// Ranking: the relevant sentence must out-score the irrelevant one.
	rel := cosSim(vecs[0], vecs[1])
	irr := cosSim(vecs[0], vecs[2])
	if rel <= irr {
		t.Fatalf("expected relevant cosine (%.4f) > irrelevant (%.4f)", rel, irr)
	}
}

func cosSim(a, b []float32) float64 {
	var dot, na, nb float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		na += float64(a[i]) * float64(a[i])
		nb += float64(b[i]) * float64(b[i])
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / (math.Sqrt(na) * math.Sqrt(nb))
}
