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

// rank returns the document positions whose BM25 score against the query is > 0,
// best first. A document with no *informative* term in common scores zero and is
// omitted — so an empty result means "this corpus has no lexical answer", which
// is a state the hook needs in order to stay silent.
func (ix *bm25Index) rank(query string) []int {
	qterms := tokenize(query)
	type scored struct {
		doc   int
		score float64
	}
	var out []scored
	for i := range ix.docTerms {
		s := ix.score(i, qterms)
		if s > 0 {
			out = append(out, scored{i, s})
		}
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

// tokenize lowercases text and splits it into alphanumeric terms. It keeps
// identifier characters together (letters + digits) and drops everything else,
// so `atl#140` becomes ["atl", "140"] and `MergedToBase` becomes
// ["mergedtobase"] — matching a query that lowercases the same way.
func tokenize(text string) []string {
	return strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
}
