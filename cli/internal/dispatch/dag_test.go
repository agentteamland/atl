package dispatch

import (
	"fmt"
	"strings"
	"testing"
)

// plan builds a well-formed plan around the units under test, filling in any
// field the DAG tests do not care about. Titles are supplied here so a DAG test
// fails for a DAG reason: Validate also rejects an empty title, since it reaches
// all three worker prompts as a blank.
func plan(units ...WorkUnit) *Plan {
	for i := range units {
		if units[i].Title == "" {
			units[i].Title = fmt.Sprintf("unit %d", units[i].ID)
		}
	}
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

// --- plan well-formedness -------------------------------------------------
//
// plan.json is written by an LLM ceremony, so every value in it is
// model-authored input rather than a programmer's literal. These pin the field
// checks that run before the DAG checks.

// The sharp one: SprintSlug reaches filepath.Join(worktreeRoot, slug, id) and
// BranchName's delivery/<slug>/<id> with no vetting of its own, so an unvetted
// slug is a path-traversal and a malformed git ref at once.
func TestValidateSprintSlugMustBePathAndRefSafe(t *testing.T) {
	unsafe := []string{
		"../escape", // climbs out of the worktree root
		"a/b",       // silently nests a directory level
		"has space", // not a valid ref component
		".hidden",   // must start alphanumeric
		"-leading",  // reads as a flag to anything shelling out
		"quote'd",
		"semi;colon",
	}
	for _, slug := range unsafe {
		p := plan(WorkUnit{ID: 1})
		p.SprintSlug = slug
		if err := Validate(p); err == nil {
			t.Errorf("sprintSlug %q accepted — it becomes a path component and a git ref", slug)
		}
	}
	for _, slug := range []string{"1", "sprint-4", "s1", "2026.08", "a_b"} {
		p := plan(WorkUnit{ID: 1})
		p.SprintSlug = slug
		if err := Validate(p); err != nil {
			t.Errorf("legitimate sprintSlug %q rejected: %v", slug, err)
		}
	}
}

func TestValidateSprintSlugRequired(t *testing.T) {
	p := plan(WorkUnit{ID: 1})
	p.SprintSlug = "  "
	if err := Validate(p); err == nil || !strings.Contains(err.Error(), "sprintSlug") {
		t.Errorf("blank sprintSlug should be rejected by name, got %v", err)
	}
}

// A zero-unit plan dispatches nothing and exits successfully — the failure mode
// that looks exactly like a clean run.
func TestValidateEmptyPlan(t *testing.T) {
	p := &Plan{SprintSlug: "s1", Granularity: GranularityPBI}
	if err := Validate(p); err == nil {
		t.Error("a plan with no units was accepted — it would report success having done nothing")
	}
}

func TestValidateGranularity(t *testing.T) {
	for _, g := range []Granularity{GranularityPBI, GranularityTask} {
		p := plan(WorkUnit{ID: 1})
		p.Granularity = g
		if err := Validate(p); err != nil {
			t.Errorf("granularity %q rejected: %v", g, err)
		}
	}
	for _, g := range []Granularity{"", "story", "PBI"} {
		p := plan(WorkUnit{ID: 1})
		p.Granularity = g
		if err := Validate(p); err == nil {
			t.Errorf("granularity %q accepted — a sprint is all-PBI or all-task", g)
		}
	}
}

// id 0 is Go's zero value, so an absent id and a real one are indistinguishable
// downstream; a negative id is not a work-item id on any backend.
func TestValidateUnitIDMustBePositive(t *testing.T) {
	for _, id := range []int{0, -1} {
		if err := Validate(plan(WorkUnit{ID: id})); err == nil {
			t.Errorf("work-unit id %d accepted", id)
		}
	}
}

func TestValidateEmptyTitle(t *testing.T) {
	p := &Plan{SprintSlug: "s1", Granularity: GranularityPBI, Units: []WorkUnit{{ID: 1, Title: "   "}}}
	if err := Validate(p); err == nil {
		t.Error("blank title accepted — it is interpolated into every worker prompt")
	}
}

func TestValidateNilPlan(t *testing.T) {
	if err := Validate(nil); err == nil {
		t.Error("nil plan accepted")
	}
}
