// Package retrieve implements atl's per-prompt hybrid knowledge retrieval.
//
// The knowledge loop's write side is automated (learning-capture -> /drain ->
// wiki + journal + agent knowledge base); #140 closes the read side by letting
// an agent consult that knowledge per prompt. Retrieval is hybrid: a lexical
// BM25 index plus this local semantic embedder, RRF-fused to a small top-k.
//
// The embedder turns text into dense vectors with a local ONNX model run
// through the pure-Go gomlx SimpleGo backend (via knights-analytics/hugot). It
// needs no native library, so the atl binary still cross-compiles to every
// release target with CGO_ENABLED=0 — the constraint that ruled out a CGo ONNX
// runtime. It is a text->vector utility, NOT an LLM: it only surfaces the right
// knowledge pages to hand the agent; Claude remains the agent.
//
// This file plus model.go is the embedder; the lexical index and fusion land in
// sibling files.
package retrieve

import (
	"context"
	"fmt"

	"github.com/knights-analytics/hugot"
	"github.com/knights-analytics/hugot/pipelines"
)

// Embedder is a reusable feature-extraction pipeline. Building it compiles the
// model graph — the cold-start cost — so reuse one Embedder across many Embed
// calls. It is not safe for concurrent use; hold one per goroutine or serialize.
type Embedder struct {
	session *hugot.Session
	pipe    *pipelines.FeatureExtractionPipeline
}

// NewEmbedder builds an embedding pipeline from a model directory holding
// model.onnx + tokenizer.json + config.json (see EnsureModel, which downloads
// and verifies them). NewGoSession selects the gomlx SimpleGo backend, keeping
// the whole path pure-Go; the caller must Close the returned Embedder.
func NewEmbedder(ctx context.Context, modelDir string) (*Embedder, error) {
	session, err := hugot.NewGoSession(ctx)
	if err != nil {
		return nil, fmt.Errorf("retrieve: new session: %w", err)
	}
	pipe, err := hugot.NewPipeline(session, hugot.FeatureExtractionConfig{
		ModelPath: modelDir,
		Name:      "atl-retrieve",
	})
	if err != nil {
		_ = session.Destroy()
		return nil, fmt.Errorf("retrieve: build pipeline from %s: %w", modelDir, err)
	}
	return &Embedder{session: session, pipe: pipe}, nil
}

// Embed returns one vector per input string, in input order. The vectors are
// deterministic (bit-exact across processes), so an index built from them is
// safely cacheable. An empty input returns no vectors.
func (e *Embedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}
	out, err := e.pipe.RunPipeline(ctx, texts)
	if err != nil {
		return nil, fmt.Errorf("retrieve: embed: %w", err)
	}
	return out.Embeddings, nil
}

// Close releases the underlying session. Safe to call on a nil-session Embedder.
func (e *Embedder) Close() error {
	if e == nil || e.session == nil {
		return nil
	}
	return e.session.Destroy()
}
