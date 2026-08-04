package retrieve

import (
	"math"
	"sort"
	"strings"
	"unicode"
)

// BM25 tuning constants (Okapi BM25 defaults). k1 controls term-frequency
// saturation; b controls length normalization.
const (
	bm25K1 = 1.2
	bm25B  = 0.75

	// bm25MaxDocRatio drops query terms present in at least this fraction of the
	// corpus. Without it the lexical arm can never come back empty: sharing the
	// word "a" with a page is enough to score above zero and rank it, so "no
	// lexical signal" — the state that lets the retriever stay silent — is
	// unreachable on any real corpus.
	//
	// Expressed as a document-frequency ratio rather than an idf threshold on
	// purpose. Idf for a term present in every document is a function of corpus
	// size (0.0123 at 40 docs, 0.0027 at 182), so a fixed idf cut means different
	// things in different projects; the ratio says exactly what is meant and holds
	// at any size. Not a stopword list either — that would be language-specific
	// and would need maintaining, and this corpus is read in two languages.
	//
	// Such terms contribute almost nothing to a score anyway, so the effect is on
	// *whether a document ranks at all*, not on the ordering of genuine matches.
	bm25MaxDocRatio = 0.9

	// bm25MinCoverage is the fraction of a query's informative terms a document
	// must contain to rank at all. It exists because the ubiquity cut above was
	// not enough to let the lexical arm come back empty: a genuinely RARE term
	// has a high idf, so one incidental match still scored above zero and kept
	// the arm alive. Measured against the live index, "ornithology taxonomy of
	// migratory seabirds in the north atlantic" ranked 5 pages.
	//
	// Coverage rather than a term count or an idf cut, because both of those were
	// measured and both fail:
	//
	//   - ">=2 matched terms" kills the case the lexical arm exists for. The
	//     single-identifier query `MergedToBase` matches exactly 1 term — the same
	//     count as the ornithology probe.
	//   - "1 term with very high idf" as an escape hatch selects BACKWARDS. That
	//     ornithology probe's best term scored idf 3.92 against MergedToBase's
	//     2.68, because idf measures rarity in the corpus, not aboutness — and the
	//     rarity of an incidental match is exactly what makes it look informative.
	//
	// Coverage handles both: a single-identifier query is 1/1 by construction, and
	// a question mostly made of words this corpus has never seen scores near zero.
	// The denominator therefore counts query terms the corpus does NOT have —
	// their absence IS the off-topic signal, and excluding them (the obvious first
	// implementation) inverts the result, scoring the ornithology probe 0.50 and
	// an offside-rule probe 1.00.
	//
	// 0.6, from 37 probes against this workspace's 327-doc index — best-doc
	// coverage by class:
	//
	//	on-topic (n=18)    0.67 … 1.00   (median 1.00)
	//	identifier (n=7)   1.00           (plus one genuinely absent from the corpus)
	//	off-topic (n=12)   0.00 … 0.67   (median 0.38)
	//
	// The bands OVERLAP at 0.67, so this is a balance and not a clean separation.
	// At 0.6: every on-topic and identifier probe survives, and 10 of 12 off-topic
	// probes go completely silent. Two still leak — "who composed the goldberg
	// variations" (0.60, 3 docs) and "which knots are strongest for climbing
	// anchors" (0.67, 1 doc). That residual is known, not overlooked.
	//
	// It cannot go higher. The arm does no stemming, so a real question loses
	// coverage to morphology alone ("swapped" does not match "swap"; the lowest
	// on-topic probe is 0.67 for exactly that reason, and this package's own
	// `rank("dispatch merge verify")` test lands at 0.667 against a document
	// reading "verifies"). 0.65 would silence one more off-topic probe and leave
	// 0.02 of margin on the on-topic side — and false silence is the worse
	// failure, since a channel that drops real answers is not worth reading either.
	bm25MinCoverage = 0.6

	// bm25MinVocab is the corpus vocabulary below which the coverage floor is not
	// applied at all.
	//
	// Coverage's denominator counts query terms the corpus lacks, which is the
	// whole point — but it only means "off-topic" if the corpus is rich enough
	// that an ordinary English question is made of words it already has. In a
	// small corpus the ubiquity cut above cannot fire either (a term must appear
	// in 90% of documents, which almost nothing does at n=3), so words like "how",
	// "does" and "on" are neither cut nor matched: they sit in the denominator and
	// silence a question the corpus genuinely answers.
	//
	// This was not predicted; a real run found it. The e2e's 3-page fixture asks
	// "how does the supervisor confirm a merge landed on the branch" of a page
	// that answers it exactly, and scored 0.44 — the floor turned a correct hit
	// into silence. Measured across every real index on the machine, the break is
	// against VOCABULARY, not document count:
	//
	//	docs  vocab   fn-words present   on-topic coverage
	//	   1    551     14 of 20          0.00
	//	   3    308     14 of 20          0.00
	//	   7     52      5 of 20          0.42
	//	  18   1905     19 of 20          0.91
	//	  31   3248     20 of 20          1.00
	//	 187  10008     20 of 20          1.00
	//	 327  10410     20 of 20          1.00
	//
	// 2000 sits above the largest corpus that failed (551) and above the one
	// borderline case (1905, still only 0.91), and below every corpus that held.
	// Under it the arm behaves exactly as it did before this floor existed —
	// which is the right default for a new project: a handful of pages is a
	// corpus with little to say, and saying a little too often is a smaller harm
	// than a knowledge base that answers nothing while it fills up.
	bm25MinVocab = 2000
)

// bm25Index is a lexical index over a document set — the half of hybrid
// retrieval that nails exact identifiers (function names, flags, `atl#140`) that
// a semantic embedder blurs. It is rebuilt in memory from the corpus (tokenizing
// tens of pages is sub-millisecond), never serialized.
type bm25Index struct {
	docTerms []map[string]int // term frequencies per doc (parallel to the corpus)
	docLen   []int            // token count per doc
	avgLen   float64
	idf      map[string]float64
	docFreq  map[string]int // documents containing each term — the ubiquity cut reads this
	numDocs  int
}

// newBM25 builds the lexical index over the given documents (same order as the
// corpus slice — scores are returned by document position).
func newBM25(docs []Doc) *bm25Index {
	ix := &bm25Index{
		docTerms: make([]map[string]int, len(docs)),
		docLen:   make([]int, len(docs)),
		idf:      map[string]float64{},
		numDocs:  len(docs),
	}
	docFreq := map[string]int{}
	var totalLen int
	for i, d := range docs {
		terms := map[string]int{}
		toks := tokenize(d.Text)
		for _, t := range toks {
			terms[t]++
		}
		ix.docTerms[i] = terms
		ix.docLen[i] = len(toks)
		totalLen += len(toks)
		for t := range terms {
			docFreq[t]++
		}
	}
	if ix.numDocs > 0 {
		ix.avgLen = float64(totalLen) / float64(ix.numDocs)
	}
	// Okapi IDF with the +1 shift that keeps common-term weights non-negative.
	for t, df := range docFreq {
		ix.idf[t] = math.Log(1 + (float64(ix.numDocs)-float64(df)+0.5)/(float64(df)+0.5))
	}
	ix.docFreq = docFreq
	return ix
}

// rank returns the document positions that both score above zero and cover at
// least bm25MinCoverage of the query's informative terms, best first. A document
// that shares only an incidental rare word is omitted — so an empty result means
// "this corpus has no lexical answer", which is a state the hook needs in order
// to stay silent.
func (ix *bm25Index) rank(query string) []int {
	qterms := tokenize(query)
	// A corpus too small to contain ordinary English cannot support the coverage
	// floor: its denominator is noise, so the floor silences questions the corpus
	// answers. Below the vocabulary threshold the arm keeps its pre-floor
	// behavior — see bm25MinVocab for the measurement.
	need := 0
	if len(ix.docFreq) >= bm25MinVocab {
		need = ix.informative(qterms)
	}
	type scored struct {
		doc   int
		score float64
	}
	var out []scored
	for i := range ix.docTerms {
		s := ix.score(i, qterms)
		if s <= 0 {
			continue
		}
		// need == 0 means every query term was cut for ubiquity, in which case
		// nothing scored above zero anyway and this is unreachable — guarded so a
		// later change to the cut can never divide by zero.
		if need > 0 && float64(ix.matched(i, qterms))/float64(need) < bm25MinCoverage {
			continue
		}
		out = append(out, scored{i, s})
	}
	sort.SliceStable(out, func(a, b int) bool { return out[a].score > out[b].score })
	docs := make([]int, len(out))
	for i, s := range out {
		docs[i] = s.doc
	}
	return docs
}

// score is the Okapi BM25 score of query terms against document i.
func (ix *bm25Index) score(i int, qterms []string) float64 {
	if ix.avgLen == 0 {
		return 0
	}
	dl := float64(ix.docLen[i])
	terms := ix.docTerms[i]
	var s float64
	for _, t := range qterms {
		tf := float64(terms[t])
		if tf == 0 {
			continue
		}
		if ix.numDocs > 0 && float64(ix.docFreq[t])/float64(ix.numDocs) >= bm25MaxDocRatio {
			continue // present in nearly every document — matching it is not signal
		}
		idf := ix.idf[t]
		s += idf * (tf * (bm25K1 + 1)) / (tf + bm25K1*(1-bm25B+bm25B*dl/ix.avgLen))
	}
	return s
}

// informative counts the DISTINCT query terms that could carry signal — every
// term except the ones cut for ubiquity. It is coverage's denominator.
//
// Terms the corpus does not contain at all are deliberately counted. They are
// unmatchable, so the intuitive move is to exclude them as "not the document's
// fault" — but that inverts the measure: a question full of words this corpus has
// never seen then looks *more* covered, because only its incidental matches
// remain in the fraction. The absence is the signal being measured.
func (ix *bm25Index) informative(qterms []string) int {
	seen := map[string]bool{}
	for _, t := range qterms {
		if seen[t] {
			continue
		}
		if ix.numDocs > 0 && float64(ix.docFreq[t])/float64(ix.numDocs) >= bm25MaxDocRatio {
			continue
		}
		seen[t] = true
	}
	return len(seen)
}

// matched counts how many DISTINCT informative query terms document i contains —
// coverage's numerator, filtered exactly as informative and score are, so the
// three cannot disagree about which terms count.
func (ix *bm25Index) matched(i int, qterms []string) int {
	seen := map[string]bool{}
	for _, t := range qterms {
		if seen[t] || ix.docTerms[i][t] == 0 {
			continue
		}
		if ix.numDocs > 0 && float64(ix.docFreq[t])/float64(ix.numDocs) >= bm25MaxDocRatio {
			continue
		}
		seen[t] = true
	}
	return len(seen)
}

// tokenize lowercases text and splits it into alphanumeric terms. It keeps
// identifier characters together (letters + digits) and drops everything else,
// so `atl#140` becomes ["atl", "140"] and `MergedToBase` becomes
// ["mergedtobase"] — matching a query that lowercases the same way.
func tokenize(text string) []string {
	return strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
}
