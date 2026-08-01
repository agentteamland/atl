package dispatch

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// slugRe is the shape a sprint slug must have to be safe as BOTH a path
// component and a git ref component: it starts alphanumeric and carries only
// alphanumerics, dot, dash and underscore. Deliberately narrower than either
// rule alone — one expression is easier to hold than two, and nothing needs
// the extra characters.
var slugRe = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

// Validate checks the plan is well-formed before any worker is spawned: the
// plan's own fields, then its dependency DAG — no duplicate ids, no self-loop,
// every predecessor references a real unit, and the graph is acyclic. A cycle or
// a dangling predecessor is a planning error the tech-lead/PM must fix — the
// scheduler REFUSES to start and surfaces it rather than silently breaking a
// link (#15 step 2, fail-fast).
//
// The field checks exist because plan.json is written by an LLM ceremony
// (/sprint-start), so every value in it is model-authored input, not a
// programmer's literal. SprintSlug is the sharp one: it reaches
// filepath.Join(worktreeRoot, slug, id) and BranchName's delivery/<slug>/<id>
// unvalidated, so a slug of "../escape" placed worktrees outside the root and a
// slug with a space produced an unusable ref. Everything else here is a plan
// that would fail confusingly deep in a run instead of loudly before it: a
// zero-unit plan dispatches nothing and reports success, a zero id collides
// with Go's zero value for "absent", and an empty title reaches all three
// worker prompts (developer, tester, tech-lead) as a blank.
func Validate(plan *Plan) error {
	if plan == nil {
		return fmt.Errorf("no plan")
	}
	if strings.TrimSpace(plan.SprintSlug) == "" {
		return fmt.Errorf("plan has no sprintSlug (it names the worktree directory and the delivery/<slug>/<id> branch)")
	}
	if !slugRe.MatchString(plan.SprintSlug) {
		return fmt.Errorf("sprintSlug %q is not path- and ref-safe: use letters, digits, dot, dash or underscore, starting alphanumeric", plan.SprintSlug)
	}
	switch plan.Granularity {
	case GranularityPBI, GranularityTask:
	default:
		return fmt.Errorf("granularity %q is neither %q nor %q (a sprint is all-PBI or all-task, never mixed)", plan.Granularity, GranularityPBI, GranularityTask)
	}
	if len(plan.Units) == 0 {
		return fmt.Errorf("plan has no units — a degenerate sprint dispatches nothing and would report success")
	}
	for _, u := range plan.Units {
		if u.ID <= 0 {
			return fmt.Errorf("work-unit id %d is not a positive work-item id", u.ID)
		}
		if strings.TrimSpace(u.Title) == "" {
			return fmt.Errorf("work-unit %d has an empty title (it is interpolated into every worker prompt)", u.ID)
		}
	}

	byID := make(map[int]WorkUnit, len(plan.Units))
	for _, u := range plan.Units {
		if _, dup := byID[u.ID]; dup {
			return fmt.Errorf("duplicate work-unit id %d", u.ID)
		}
		byID[u.ID] = u
	}
	for _, u := range plan.Units {
		for _, p := range u.Predecessors {
			if p == u.ID {
				return fmt.Errorf("work-unit %d lists itself as a predecessor", u.ID)
			}
			if _, ok := byID[p]; !ok {
				return fmt.Errorf("work-unit %d has predecessor %d not present in the plan", u.ID, p)
			}
		}
	}
	if cyc := findCycle(plan, byID); cyc != nil {
		return fmt.Errorf("dependency cycle: %s", chainString(cyc))
	}
	return nil
}

// Ready returns the plan's ready frontier: units that are not yet done and all
// of whose predecessors are done, ordered by admission priority — ascending
// StackRank (Azure backlog rank; a lower value is higher priority, so the PO's
// top-ranked ready work is admitted first), ties broken by id for determinism.
// It considers neither the concurrency cap nor which units are already running;
// the scheduler layers those on (#15 step 3).
func Ready(plan *Plan, done map[int]bool) []WorkUnit {
	var ready []WorkUnit
	for _, u := range plan.Units {
		if done[u.ID] {
			continue
		}
		if allDone(u.Predecessors, done) {
			ready = append(ready, u)
		}
	}
	sort.Slice(ready, func(i, j int) bool {
		if ready[i].StackRank != ready[j].StackRank {
			return ready[i].StackRank < ready[j].StackRank
		}
		return ready[i].ID < ready[j].ID
	})
	return ready
}

func allDone(preds []int, done map[int]bool) bool {
	for _, p := range preds {
		if !done[p] {
			return false
		}
	}
	return true
}

// findCycle returns a dependency cycle as a chain of ids (a -> b -> … -> a), or
// nil if the DAG is acyclic. Edges follow "depends-on": a unit points to each of
// its predecessors. Entry points are visited in ascending id order so the
// reported chain is stable.
func findCycle(plan *Plan, byID map[int]WorkUnit) []int {
	const (
		white = 0
		gray  = 1
		black = 2
	)
	color := make(map[int]int, len(plan.Units))
	var stack []int
	var dfs func(id int) []int
	dfs = func(id int) []int {
		color[id] = gray
		stack = append(stack, id)
		for _, p := range byID[id].Predecessors {
			switch color[p] {
			case gray:
				// Back-edge to p, which is already on the stack: close the cycle
				// from p's position through the current node and back to p.
				for i, n := range stack {
					if n == p {
						return append(append([]int(nil), stack[i:]...), p)
					}
				}
			case white:
				if cyc := dfs(p); cyc != nil {
					return cyc
				}
			}
		}
		stack = stack[:len(stack)-1]
		color[id] = black
		return nil
	}
	for _, id := range sortedIDs(plan.Units) {
		if color[id] == white {
			if cyc := dfs(id); cyc != nil {
				return cyc
			}
		}
	}
	return nil
}

func sortedIDs(units []WorkUnit) []int {
	ids := make([]int, len(units))
	for i, u := range units {
		ids[i] = u.ID
	}
	sort.Ints(ids)
	return ids
}

func chainString(ids []int) string {
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = strconv.Itoa(id)
	}
	return strings.Join(parts, " -> ")
}
