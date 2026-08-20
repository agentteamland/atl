package retrieve

import (
	"context"
	"testing"
)

// TestBuildIncrementalStopsCallingTheEmbedderOnceTheContextIsDone pins the
// second half of atl#608: a build whose deadline has passed must stop, not spend
// its remaining documents calling an embedder that can only fail.
//
// Without the check the loop runs to the end of the corpus against a dead
// context. Each call errors, falls into the per-chunk retry (which errors too),
// pools an empty slice and stores nil — so a cancelled build keeps working for
// however long it has left and produces nothing. The resilient fallback is what
// makes that waste silent: it turns a hard cancellation into an empty result.
//
// The instrument is a zero-value Embedder, and it is chosen so the property has
// a WITNESS. The behaviour under test is "makes no further embed calls", and the
// vectors cannot show it — with or without the fix every entry ends up nil, so
// output alone cannot separate "stopped" from "called and got nothing". A zero
// Embedder has no pipeline, so any call at all is observable. Nothing here needs
// the model downloaded, which is what lets it run on CI.
func TestBuildIncrementalStopsCallingTheEmbedderOnceTheContextIsDone(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // the deadline has already passed

	docs := []Doc{
		{Path: "a", Text: "the first page, which the build never reaches"},
		{Path: "b", Text: "the second page, likewise"},
	}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("the build called the embedder after its context was done (%v); on a real "+
				"embedder that is not a panic but a long silent stretch of failing calls, each one "+
				"storing a nil vector that the (path, text) reuse key then makes permanent", r)
		}
	}()

	ix := BuildIncremental(ctx, docs, &Embedder{}, nil)

	if len(ix.Vecs) != len(docs) {
		t.Fatalf("Vecs must stay parallel to Docs even on a cancelled build: %d vecs for %d docs",
			len(ix.Vecs), len(docs))
	}
	for i, v := range ix.Vecs {
		if v != nil {
			t.Fatalf("doc %d carries a vector from a build that never ran", i)
		}
	}
}
