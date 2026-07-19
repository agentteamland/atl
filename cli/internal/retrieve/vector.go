package retrieve

import (
	"math"
	"sort"
)

// cosine is the cosine similarity of two equal-length vectors, in [-1, 1]. It
// returns 0 for a zero-magnitude or length-mismatched vector (defensive: a
// corrupt index should degrade to "no semantic signal", not panic).
func cosine(a, b []float32) float64 {
	if len(a) != len(b) {
		return 0
	}
	var dot, na, nb float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		na += float64(a[i]) * float64(a[i])
		nb += float64(b[i]) * float64(b[i])
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / (math.Sqrt(na) * math.Sqrt(nb))
}

// semanticRank returns document positions ordered by cosine similarity to the
// query vector, best first — a brute-force scan (trivial at the tens-of-pages
// corpus scale). Every document appears, since every one has a similarity.
func semanticRank(query []float32, vecs [][]float32) []int {
	type scored struct {
		doc int
		sim float64
	}
	scores := make([]scored, len(vecs))
	for i, v := range vecs {
		scores[i] = scored{i, cosine(query, v)}
	}
	sort.SliceStable(scores, func(a, b int) bool { return scores[a].sim > scores[b].sim })
	docs := make([]int, len(scores))
	for i, s := range scores {
		docs[i] = s.doc
	}
	return docs
}
