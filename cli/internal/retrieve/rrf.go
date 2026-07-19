package retrieve

import "sort"

// rrfK is the Reciprocal Rank Fusion constant. 60 is the value from the
// original RRF paper (Cormack et al.); it damps the influence of top ranks just
// enough that a document strong in one ranker can still be lifted by a weak
// showing in the other.
const rrfK = 60

// rrf fuses ranked lists of document positions (each best-first) into one
// ranking by reciprocal rank: a document's fused score is the sum over lists of
// 1/(rrfK + rank), where rank is its 0-based position in that list. A document
// absent from a list contributes nothing from it. This lets BM25 (exact
// identifiers) and semantic similarity (concepts) reinforce each other without
// having to reconcile their incomparable raw scores. Returns document positions
// best-first.
func rrf(lists ...[]int) []int {
	fused := map[int]float64{}
	for _, list := range lists {
		for rank, doc := range list {
			fused[doc] += 1.0 / float64(rrfK+rank)
		}
	}
	docs := make([]int, 0, len(fused))
	for doc := range fused {
		docs = append(docs, doc)
	}
	sort.SliceStable(docs, func(a, b int) bool {
		if fused[docs[a]] != fused[docs[b]] {
			return fused[docs[a]] > fused[docs[b]]
		}
		return docs[a] < docs[b] // stable tie-break by document position
	})
	return docs
}
