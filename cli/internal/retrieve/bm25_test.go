package retrieve

import (
	"fmt"
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

// A query that shares only ubiquitous words with the corpus must rank nothing.
// Before the idf cut, sharing "the" or "a" was enough to score above zero, so
// the lexical arm always returned something and the retriever could never be
// silent — which is the state the hook needs in order to say "no signal".
func TestBM25IgnoresTermsPresentInNearlyEveryDoc(t *testing.T) {
	docs := make([]Doc, 40)
	for i := range docs {
		docs[i] = Doc{
			Path: fmt.Sprintf("p%02d.md", i),
			Text: fmt.Sprintf("the system has a component number %d for the operator", i),
		}
	}
	docs[7].Text = "the system has a zeppelin component for the operator"
	ix := newBM25(docs)

	if got := ix.rank("the a for has"); len(got) != 0 {
		t.Errorf("a stopword-only query must rank nothing, got %d docs", len(got))
	}
	// …and a term that is genuinely rare still ranks, so the cut has not simply
	// disabled the lexical arm.
	got := ix.rank("zeppelin")
	if len(got) != 1 || got[0] != 7 {
		t.Errorf("a rare term must still rank its document, got %v", got)
	}
}
