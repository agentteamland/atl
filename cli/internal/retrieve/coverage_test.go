package retrieve

import (
	"fmt"
	"testing"
)

// corpusOf builds a corpus large enough that the ubiquity cut behaves as it does
// in production (a term must appear in >=90% of documents to be cut), with the
// caller's documents at the front.
func corpusOf(t *testing.T, front ...string) []Doc {
	t.Helper()
	docs := make([]Doc, 0, 40)
	for _, text := range front {
		docs = append(docs, Doc{Path: fmt.Sprintf("front%02d.md", len(docs)), Text: text})
	}
	for i := len(docs); i < 40; i++ {
		docs = append(docs, Doc{
			Path: fmt.Sprintf("filler%02d.md", i),
			Text: fmt.Sprintf("an unrelated page about widget number %d and its assembly", i),
		})
	}
	return docs
}

// The defect: a rare term has a high idf, so ONE incidental match kept the
// lexical arm alive and the retriever could never be silent on a question this
// corpus has no answer to. Measured live, an ornithology question ranked 5 pages.
func TestOffTopicQueryWithOneIncidentalMatchRanksNothing(t *testing.T) {
	ix := newBM25(corpusOf(t,
		"the dispatch engine verifies a git merge against the durable branch state",
		"tidal charts for the north atlantic shipping lanes",
	))

	// "atlantic" is genuinely rare here, so it scores well on its own — which is
	// exactly the coincidence the floor exists to reject.
	got := ix.rank("ornithology taxonomy of migratory seabirds in the north atlantic")
	if len(got) != 0 {
		t.Fatalf("an off-topic query sharing one rare word must rank nothing, got %v", got)
	}
}

// The case the lexical arm exists for, and the one a raw ">=2 matched terms"
// rule would have killed: a single identifier covers 1/1 of its query.
func TestSingleIdentifierQueryStillRanks(t *testing.T) {
	ix := newBM25(corpusOf(t,
		"MergedToBase reads the durable git state rather than the worker exit code",
	))
	got := ix.rank("MergedToBase")
	if len(got) != 1 || got[0] != 0 {
		t.Fatalf("a single-identifier query must still rank its document, got %v", got)
	}
}

// A real multi-word question whose answer is genuinely present must survive.
func TestOnTopicQueryStillRanksDespiteTheFloor(t *testing.T) {
	ix := newBM25(corpusOf(t,
		"the promotion gate binds the approval record to the promoted commit sha",
	))
	got := ix.rank("how is the promotion gate bound to the promoted commit")
	if len(got) == 0 || got[0] != 0 {
		t.Fatalf("an on-topic question must rank its answer, got %v", got)
	}
}

// The floor is a ratio, not a count: the same single match is enough for a
// one-term query and not enough for a long one. Without that, either identifiers
// break or off-topic questions keep ranking — the two failures the constant's
// comment records as the reason coverage was chosen.
func TestCoverageIsARatioNotACount(t *testing.T) {
	ix := newBM25(corpusOf(t, "zeppelin"))

	if got := ix.rank("zeppelin"); len(got) != 1 {
		t.Fatalf("1 of 1 informative terms must rank, got %v", got)
	}
	if got := ix.rank("zeppelin airship hangar mooring mast helium"); len(got) != 0 {
		t.Fatalf("1 of 6 informative terms must not rank, got %v", got)
	}
}

// Query terms the corpus has never seen must stay in the denominator. Excluding
// them is the obvious first implementation and it INVERTS the measure — an
// off-topic question then looks more covered, because only its incidental
// matches remain in the fraction.
func TestUnseenQueryTermsCountAgainstCoverage(t *testing.T) {
	ix := newBM25(corpusOf(t, "the widget calibration procedure"))

	// One matched term ("calibration") against five terms the corpus lacks. If
	// unseen terms were dropped from the denominator this would be 1/1 and rank.
	got := ix.rank("calibration of quattrocento fresco pigment chemistry florence")
	if len(got) != 0 {
		t.Fatalf("unseen query terms must count against coverage, got %v", got)
	}
}

// A stopword-only query has no informative terms at all. The guard must not
// divide by zero, and the result must stay empty.
func TestAllUbiquitousQueryStaysEmptyAndDoesNotPanic(t *testing.T) {
	docs := make([]Doc, 40)
	for i := range docs {
		docs[i] = Doc{Path: fmt.Sprintf("p%02d.md", i), Text: "the system has a component for the operator"}
	}
	ix := newBM25(docs)
	if got := ix.rank("the a for has"); len(got) != 0 {
		t.Fatalf("a query of only ubiquitous terms must rank nothing, got %v", got)
	}
}

// Coverage must be computed over DISTINCT terms on BOTH sides of the fraction.
// The repetition counts here are chosen to straddle the floor: a smaller number
// of repeats leaves the ratio above 0.6 either way, so the test passes against a
// broken implementation and proves nothing.
func TestRepeatedQueryTermsDoNotChangeCoverage(t *testing.T) {
	ix := newBM25(corpusOf(t, "the promotion gate binds the approval record to the commit"))

	// Denominator: 4 distinct terms, all matched. A raw token count would make it
	// 4/12 = 0.33 and drop a document that answers the question exactly.
	plain := ix.rank("promotion gate approval commit")
	repeated := ix.rank("promotion promotion promotion gate gate gate approval approval approval commit commit commit")
	if len(plain) == 0 || len(plain) != len(repeated) {
		t.Fatalf("repeating a query term changed the ranked set: %v vs %v", plain, repeated)
	}

	// Numerator: one matched term repeated must not buy coverage it did not earn.
	// Counted per occurrence this is 4/6 = 0.67 and would rank; distinct, it is
	// 1/6 = 0.17 and must not.
	ix2 := newBM25(corpusOf(t, "zeppelin"))
	if got := ix2.rank("zeppelin zeppelin zeppelin zeppelin airship hangar mooring mast"); len(got) != 0 {
		t.Fatalf("repeating one matched term must not lift coverage over the floor, got %v", got)
	}
}
