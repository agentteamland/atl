package retrieve

import (
	"math"
	"reflect"
	"testing"
)

func TestCosine(t *testing.T) {
	if got := cosine([]float32{1, 0}, []float32{1, 0}); math.Abs(got-1) > 1e-9 {
		t.Fatalf("identical vectors: want 1, got %v", got)
	}
	if got := cosine([]float32{1, 0}, []float32{0, 1}); math.Abs(got) > 1e-9 {
		t.Fatalf("orthogonal vectors: want 0, got %v", got)
	}
	if got := cosine([]float32{1, 2, 3}, []float32{1, 2}); got != 0 {
		t.Fatalf("length mismatch: want 0, got %v", got)
	}
	if got := cosine([]float32{0, 0}, []float32{1, 1}); got != 0 {
		t.Fatalf("zero magnitude: want 0, got %v", got)
	}
}

func TestSemanticRankOrdersByCloseness(t *testing.T) {
	query := []float32{1, 0}
	vecs := [][]float32{
		{0, 1},     // doc 0: orthogonal (far)
		{1, 0},     // doc 1: identical (closest)
		{0.7, 0.7}, // doc 2: 45 degrees (middle)
	}
	got := semanticRank(query, vecs)
	want := []int{1, 2, 0}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("semanticRank: got %v want %v", got, want)
	}
}

func TestRRFReinforcesAndFallsBack(t *testing.T) {
	// Doc 2 is top of the lexical list and second in the semantic list; doc 0 is
	// top semantic but absent lexically. Doc 2's presence in both should lift it
	// above doc 0.
	lex := []int{2, 3}
	sem := []int{0, 2, 1}
	got := rrf(lex, sem)
	if got[0] != 2 {
		t.Fatalf("expected doc 2 (strong in both) ranked first, got %v", got)
	}

	// A nil ranker (no semantic signal) degrades to the other list's order.
	if got := rrf([]int{5, 6, 7}, nil); !reflect.DeepEqual(got, []int{5, 6, 7}) {
		t.Fatalf("BM25-only fallback: got %v want [5 6 7]", got)
	}

	// Both empty -> nothing.
	if got := rrf(nil, nil); len(got) != 0 {
		t.Fatalf("empty fusion: got %v", got)
	}
}
