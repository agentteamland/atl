package dispatch

import (
	"strings"
	"testing"
)

func plan(units ...WorkUnit) *Plan {
	return &Plan{SprintSlug: "s1", Granularity: GranularityPBI, Units: units}
}

func TestValidateAcyclic(t *testing.T) {
	p := plan(
		WorkUnit{ID: 1, Predecessors: nil},
		WorkUnit{ID: 2, Predecessors: []int{1}},
		WorkUnit{ID: 3, Predecessors: []int{1, 2}},
	)
	if err := Validate(p); err != nil {
		t.Errorf("valid DAG rejected: %v", err)
	}
}

func TestValidateCycle(t *testing.T) {
	p := plan(
		WorkUnit{ID: 1, Predecessors: []int{3}},
		WorkUnit{ID: 2, Predecessors: []int{1}},
		WorkUnit{ID: 3, Predecessors: []int{2}},
	)
	err := Validate(p)
	if err == nil {
		t.Fatal("cycle 1->3->2->1 not detected")
	}
	if !strings.Contains(err.Error(), "cycle") {
		t.Errorf("error should name the cycle: %v", err)
	}
	// Every id in the cycle should appear in the surfaced chain.
	for _, id := range []string{"1", "2", "3"} {
		if !strings.Contains(err.Error(), id) {
			t.Errorf("cycle chain %q missing id %s", err.Error(), id)
		}
	}
}

func TestValidateSelfLoop(t *testing.T) {
	if err := Validate(plan(WorkUnit{ID: 5, Predecessors: []int{5}})); err == nil {
		t.Error("self-predecessor not detected")
	}
}

func TestValidateDanglingPredecessor(t *testing.T) {
	if err := Validate(plan(WorkUnit{ID: 1, Predecessors: []int{99}})); err == nil {
		t.Error("dangling predecessor 99 not detected")
	}
}

func TestValidateDuplicateID(t *testing.T) {
	if err := Validate(plan(WorkUnit{ID: 1}, WorkUnit{ID: 1})); err == nil {
		t.Error("duplicate id not detected")
	}
}

func TestReadyPredecessorGating(t *testing.T) {
	p := plan(
		WorkUnit{ID: 1, StackRank: 1},
		WorkUnit{ID: 2, StackRank: 2, Predecessors: []int{1}},
	)
	// Nothing done: only 1 is ready (2 waits on 1).
	got := Ready(p, map[int]bool{})
	if len(got) != 1 || got[0].ID != 1 {
		t.Fatalf("with nothing done, ready = %+v, want [1]", got)
	}
	// 1 done: 2 becomes ready, 1 is excluded (already done).
	got = Ready(p, map[int]bool{1: true})
	if len(got) != 1 || got[0].ID != 2 {
		t.Fatalf("with 1 done, ready = %+v, want [2]", got)
	}
}

func TestReadyStackRankOrder(t *testing.T) {
	// All roots; lower StackRank must come first, ties broken by id.
	p := plan(
		WorkUnit{ID: 10, StackRank: 3},
		WorkUnit{ID: 11, StackRank: 1},
		WorkUnit{ID: 12, StackRank: 1},
		WorkUnit{ID: 13, StackRank: 2},
	)
	got := Ready(p, map[int]bool{})
	want := []int{11, 12, 13, 10} // rank 1 (id 11,12), rank 2 (13), rank 3 (10)
	if len(got) != len(want) {
		t.Fatalf("ready len = %d, want %d", len(got), len(want))
	}
	for i, id := range want {
		if got[i].ID != id {
			t.Errorf("ready[%d].ID = %d, want %d (order %+v)", i, got[i].ID, id, got)
		}
	}
}
