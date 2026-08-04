package retrieve

import (
	"fmt"
	"testing"
)

// corpusOf builds a corpus that behaves like a real one on both counts the
// coverage floor depends on: the ubiquity cut can actually fire (the filler
// shares a frame, so its function words appear in ~every document), and the
// vocabulary clears bm25MinVocab so the floor is applied at all.
//
// Both halves are load-bearing. A corpus of a few short documents leaves
// ordinary English in the denominator with nothing to match — which is the
// production failure bm25MinVocab exists for, and it would make every test here
// pass for the wrong reason: the floor would simply never run.
func corpusOf(t *testing.T, front ...string) []Doc {
	t.Helper()
	docs := make([]Doc, 0, 64)
	for _, text := range front {
		docs = append(docs, Doc{Path: fmt.Sprintf("front%02d.md", len(docs)), Text: text})
	}
	for i := len(docs); i < 64; i++ {
		// A shared natural-language frame (so "the", "a", "how", "does", "on" are
		// ubiquitous and get cut) plus 40 terms unique to this page (so the
		// vocabulary grows past bm25MinVocab).
		text := "how does the operator handle a request on the system when it arrives"
		for j := 0; j < 40; j++ {
			text += fmt.Sprintf(" topic%03dterm%02d", i, j)
		}
		docs = append(docs, Doc{Path: fmt.Sprintf("filler%02d.md", i), Text: text})
	}
	return docs
}

// The fixtures must exercise the floor, not sneak under it. Without this, a
// later edit that shrinks the corpus would turn every test below into a silent
// no-op that still reports PASS.
func TestCoverageFixtureActuallyEngagesTheFloor(t *testing.T) {
	ix := newBM25(corpusOf(t, "the promotion gate binds the approval record to the commit"))
	if len(ix.docFreq) < bm25MinVocab {
		t.Fatalf("fixture vocabulary %d is below bm25MinVocab %d — the floor would never run and these tests would pass vacuously",
			len(ix.docFreq), bm25MinVocab)
	}
	if ix.docFreq["the"] == 0 || float64(ix.docFreq["the"])/float64(ix.numDocs) < bm25MaxDocRatio {
		t.Fatalf("fixture does not make function words ubiquitous (the: %d/%d) — the denominator would not behave as production's does",
			ix.docFreq["the"], ix.numDocs)
	}
}

// Below the vocabulary threshold the floor must not apply. This is the e2e's
// 3-page fixture, and the exact query a real run found the floor silencing: the
// page answers it, and the answer must come back.
func TestASmallCorpusKeepsItsPreFloorBehavior(t *testing.T) {
	ix := newBM25([]Doc{
		{Path: "merge-verify.md", Text: "Verify durable state not worker exit-code. A deterministic supervisor confirms a git merge by reading the durable branch state, never trusting an LLM worker exit code."},
		{Path: "pr-merge.md", Text: "PR merge discipline. Never merge pull requests from Claude; surface the URL and stop."},
		{Path: "architecture.md", Text: "Architecture store. The GitHub backend keeps its durable knowledge in the in-repo docs tree."},
	})
	if len(ix.docFreq) >= bm25MinVocab {
		t.Fatalf("this fixture is meant to be below the vocabulary threshold, got %d", len(ix.docFreq))
	}
	got := ix.rank("how does the supervisor confirm a merge landed on the branch")
	if len(got) == 0 || got[0] != 0 {
		t.Fatalf("a small corpus must still answer a question it covers — coverage here is 0.44 and the floor must not apply; got %v", got)
	}
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
