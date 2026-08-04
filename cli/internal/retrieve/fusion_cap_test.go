package retrieve

import "testing"

// The measured defect: the lexical arm ranked the answer FIRST and the fused
// top-5 did not contain it, because the semantic arm returned four times as many
// documents and its TAIL overlapped the lexical list. Every overlapping page
// collected a second contribution the answer never got, and five of them
// outscored it.
//
// The shape matters and is easy to model wrongly: the overlap is in the semantic
// arm's tail, not its head. A first draft of this test put the overlap at the
// top, where the cap changes nothing — and it correctly failed.
func lexAndSemantic() (lexical, semantic []int) {
	lexical = make([]int, 16) // answer at 0, then 1..15
	for i := range lexical {
		lexical[i] = i
	}
	semantic = make([]int, 0, 25)
	for i := 0; i < 10; i++ {
		semantic = append(semantic, 200+i) // head: no overlap, survives the cap
	}
	for i := 1; i <= 15; i++ {
		semantic = append(semantic, i) // tail: overlaps the lexical arm
	}
	return
}

func TestALexicalFirstPlaceSurvivesALongSemanticArm(t *testing.T) {
	lexical, semantic := lexAndSemantic()
	got := fuseArms(lexical, semantic, 5)
	if len(got) == 0 || got[0] != 0 {
		t.Fatalf("the answer the lexical arm ranked first is not first in the fusion: %v", got)
	}
}

// Without the cap the same input loses it entirely — not just demoted, absent
// from the top-5. This is what shipped and what the recall measurement caught;
// it guards the cap against removal as a "simplification".
func TestWithoutTheCapTheLongArmsTailWins(t *testing.T) {
	lexical, semantic := lexAndSemantic()
	uncapped := rrf(lexical, semantic)
	if len(uncapped) > 5 {
		uncapped = uncapped[:5]
	}
	for _, d := range uncapped {
		if d == 0 {
			t.Fatalf("uncapped fusion kept the answer, so this fixture no longer reproduces the defect: %v", uncapped)
		}
	}
}

// The cap must not truncate a semantic arm that is already short, and must never
// drop the lexical arm.
func TestFuseArmsLeavesShortArmsAlone(t *testing.T) {
	lex := []int{3, 1}
	sem := []int{1, 2}
	got := fuseArms(lex, sem, 0)
	seen := map[int]bool{}
	for _, d := range got {
		seen[d] = true
	}
	for _, d := range append(append([]int{}, lex...), sem...) {
		if !seen[d] {
			t.Fatalf("document %d disappeared from the fusion: %v", d, got)
		}
	}
}

// A semantic arm longer than the cap contributes exactly semanticFusionCap
// documents — no more, and the ones it keeps are its best.
func TestFuseArmsKeepsTheSemanticArmsBest(t *testing.T) {
	sem := make([]int, 50)
	for i := range sem {
		sem[i] = 1000 + i
	}
	got := fuseArms(nil, sem, 0)
	if len(got) != semanticFusionCap {
		t.Fatalf("semantic-only fusion returned %d documents, want %d", len(got), semanticFusionCap)
	}
	if got[0] != 1000 {
		t.Fatalf("kept the wrong end of the semantic arm: %v", got[:3])
	}
}
