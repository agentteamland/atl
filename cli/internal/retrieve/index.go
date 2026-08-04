package retrieve

import (
	"context"
	"encoding/gob"
	"math"
	"os"
	"path/filepath"
)

// indexFormatVersion bumps when the on-disk gob layout OR the embedding model
// changes — the incremental reuse key is (path, text), which is blind to the
// model, so swapping models without a bump would produce a silently mixed index
// (old-model vectors on unchanged docs, new-model on changed ones). The index is
// a rebuildable cache, so a version mismatch is treated as "no index yet" (a full
// rebuild) rather than an error.
//
//	1 -> 2: the embedder moved from all-MiniLM-L6-v2 (English-only) to
//	        paraphrase-multilingual-MiniLM-L12-v2. Same 384 dimensions, so the
//	        gob layout is untouched and only the vectors' meaning changed — which
//	        is exactly the case an unbumped version would hide, leaving a silently
//	        mixed index that ranks old and new pages against each other.
const indexFormatVersion = 2

// Index is the persisted retrieval index for one corpus: the documents plus one
// embedding vector per document. BM25 is rebuilt in memory from the documents at
// query time (tokenizing tens of pages is sub-millisecond), so only the
// documents and their vectors are serialized.
type Index struct {
	Version int
	Docs    []Doc
	Vecs    [][]float32 // parallel to Docs; empty when the corpus is empty
}

// Result is one retrieved document, best-first from Query.
type Result struct {
	Path  string
	Title string
}

// Build indexes docs from scratch — see BuildIncremental, which it delegates to
// with no prior index.
func Build(ctx context.Context, docs []Doc, e *Embedder) (*Index, error) {
	return BuildIncremental(ctx, docs, e, nil), nil
}

// BuildIncremental indexes docs, producing one vector per document (see embedDoc
// for how a long document is chunked and pooled), reusing the embedding of any
// document whose path and text are unchanged from old. That reuse is what makes
// re-indexing cheap: a drain that touched one page re-embeds only that page
// (seconds) instead of the whole corpus (minutes). It is best-effort — a document
// that can't be embedded gets a nil vector and stays lexically searchable. A nil
// embedder builds a lexical-only index (no vectors) so retrieval works before the
// model has downloaded, degrading to BM25. This is the cold path — run at drain
// time, not per query.
func BuildIncremental(ctx context.Context, docs []Doc, e *Embedder, old *Index) *Index {
	ix := &Index{Version: indexFormatVersion, Docs: docs}
	if e == nil {
		return ix // lexical-only index
	}
	reuse := make(map[string][]float32, len(docs))
	if old != nil && len(old.Vecs) == len(old.Docs) {
		for i, d := range old.Docs {
			reuse[d.Path+"\x00"+d.Text] = old.Vecs[i]
		}
	}
	ix.Vecs = make([][]float32, len(docs))
	for i, d := range docs {
		if v, ok := reuse[d.Path+"\x00"+d.Text]; ok {
			ix.Vecs[i] = v // unchanged since the last index — reuse the embedding
			continue
		}
		ix.Vecs[i] = embedDoc(ctx, e, d.Text)
	}
	return ix
}

// embedDoc turns a document into one vector. The embedder has a hard 512-token
// limit and does not truncate, and most real pages exceed it, so the text is
// split into token-safe chunks, each L2-normalized, then mean-pooled into a
// single document vector (BM25 covers the full text for exact terms; the mean
// captures the page's overall topic). All of a document's chunks are embedded in
// one batch call for speed; if that batch fails (a pathological chunk), it falls
// back to per-chunk embedding so one bad chunk can't sink the document. A
// document with no usable chunk returns a nil vector (still lexically searchable).
func embedDoc(ctx context.Context, e *Embedder, text string) []float32 {
	chunks := chunkText(text)
	if len(chunks) == 0 {
		return nil
	}
	vecs, err := e.Embed(ctx, chunks)
	if err != nil {
		vecs = embedPerChunk(ctx, e, chunks) // resilient fallback
	}
	return meanPool(vecs)
}

// embedPerChunk embeds each chunk on its own, dropping any that fail — the
// resilient path when a whole-document batch errors.
func embedPerChunk(ctx context.Context, e *Embedder, chunks []string) [][]float32 {
	var out [][]float32
	for _, c := range chunks {
		if v, err := e.Embed(ctx, []string{c}); err == nil {
			out = append(out, v[0])
		}
	}
	return out
}

// meanPool L2-normalizes each vector (so each contributes equally) and averages
// them. Returns nil for no vectors.
func meanPool(vecs [][]float32) []float32 {
	if len(vecs) == 0 {
		return nil
	}
	var sum []float32
	for _, v := range vecs {
		normalize(v)
		if sum == nil {
			sum = make([]float32, len(v))
		}
		for j := range v {
			sum[j] += v[j]
		}
	}
	for j := range sum {
		sum[j] /= float32(len(vecs))
	}
	return sum
}

// normalize scales v to unit L2 length in place (a no-op for a zero vector) so
// each chunk contributes equally to the mean pool.
func normalize(v []float32) {
	var norm float64
	for _, x := range v {
		norm += float64(x) * float64(x)
	}
	if norm == 0 {
		return
	}
	inv := float32(1 / math.Sqrt(norm))
	for i := range v {
		v[i] *= inv
	}
}

// Query returns the top-k documents for a prompt, fusing a BM25 lexical ranking
// with a semantic (cosine) ranking via RRF. It embeds the prompt once. When the
// index carries no vectors (an embedder-less build), it degrades to BM25 only.
// MinSimDefault is the cosine floor the semantic arm applies, below which a
// document is treated as "no signal" rather than as a weak match.
//
// Derived, not guessed — 13 probes against this workspace's live 182-doc index,
// top-1 cosine by band:
//
//	on-topic English   0.298 … 0.631   (mean 0.451)
//	off-topic English  0.097 … 0.227   (mean 0.152)
//	machine / 1-word   0.204 … 0.426
//	on-topic Turkish   0.107 … 0.175   (mean 0.140)
//
// 0.25 separates on-topic English from everything else it can separate. The
// fourth row is the uncomfortable one and must not be filed as a tuning detail:
// an on-topic question in the user's own language scores BELOW off-topic English,
// so this model has no signal there at all. The same question in both languages
// measured 0.631 (en) against 0.138 (tr). Consequence to hold onto: under this
// floor a non-English prompt correctly yields nothing — honest, and the reason
// the multilingual model is a fix rather than an optimisation.
const MinSimDefault = 0.25

// Query ranks the corpus against prompt and returns up to k results. minSim is
// the semantic arm's cosine floor (MinSimDefault unless a caller is measuring);
// the lexical arm's floor is inherent — BM25 already drops any document sharing
// no query term. When both arms come back empty the result is empty, which is
// the point: the retriever must be able to say it has nothing.
func (ix *Index) Query(ctx context.Context, prompt string, e *Embedder, k int, minSim float64) ([]Result, error) {
	if len(ix.Docs) == 0 || k <= 0 {
		return nil, nil
	}
	lexical := newBM25(ix.Docs).rank(prompt)

	var semantic []int
	if e != nil && len(ix.Vecs) == len(ix.Docs) {
		// A failed or empty query embed degrades to BM25-only rather than failing
		// the query — retrieval stays useful (and, for the fail-open hook, alive).
		if qv, err := e.Embed(ctx, []string{prompt}); err == nil && len(qv) > 0 {
			semantic = semanticRank(qv[0], ix.Vecs, minSim)
		}
	}

	fused := rrf(lexical, semantic)
	if len(fused) > k {
		fused = fused[:k]
	}
	out := make([]Result, len(fused))
	for i, doc := range fused {
		out[i] = Result{Path: ix.Docs[doc].Path, Title: ix.Docs[doc].Title}
	}
	return out, nil
}

// Save writes the index to path via a same-dir temp file + atomic rename, so a
// concurrent reader never sees a half-written index.
func (ix *Index) Save(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".retrieve-index-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op once the rename succeeds
	if err := gob.NewEncoder(tmp).Encode(ix); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

// Load reads an index from path. A missing, corrupt, or wrong-version file
// returns (nil, nil) — "no usable index", since the index is a rebuildable cache
// and the caller (fail-open) simply skips injection until the next rebuild.
func Load(path string) (*Index, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()
	var ix Index
	if err := gob.NewDecoder(f).Decode(&ix); err != nil {
		return nil, nil // corrupt cache -> treat as absent, rebuild later
	}
	if ix.Version != indexFormatVersion {
		return nil, nil // stale format -> rebuild later
	}
	return &ix, nil
}
