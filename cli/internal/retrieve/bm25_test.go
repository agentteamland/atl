package retrieve

import (
	"reflect"
	"testing"
)

func TestTokenize(t *testing.T) {
	got := tokenize("atl#140 MergedToBase dispatch, merge-verify!")
	want := []string{"atl", "140", "mergedtobase", "dispatch", "merge", "verify"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("tokenize:\n got %v\nwant %v", got, want)
	}
}

func TestBM25RanksOverlapFirstAndOmitsNonMatches(t *testing.T) {
	docs := []Doc{
		{Path: "a", Text: "the dispatch engine verifies a git merge against the durable branch state"},
		{Path: "b", Text: "profile facts are gated by privacy tier before they are written"},
		{Path: "c", Text: "a banana bread recipe with very ripe bananas and cinnamon"},
	}
	ix := newBM25(docs)
	got := ix.rank("dispatch merge verify")

	if len(got) == 0 || got[0] != 0 {
		t.Fatalf("expected doc 0 (dispatch/merge) ranked first, got %v", got)
	}
	// Docs b and c share no query term, so they must be omitted (rank returns
	// only positive-scoring docs).
	for _, d := range got {
		if d == 1 || d == 2 {
			t.Fatalf("non-matching doc %d should be omitted, got ranking %v", d, got)
		}
	}
}

func TestBM25EmptyCorpus(t *testing.T) {
	ix := newBM25(nil)
	if got := ix.rank("anything"); len(got) != 0 {
		t.Fatalf("empty corpus should rank nothing, got %v", got)
	}
}
