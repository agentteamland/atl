package dispatch

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// Validate checks the plan's dependency DAG is well-formed before any worker is
// spawned: no duplicate ids, no self-loop, every predecessor references a real
// unit, and the graph is acyclic. A cycle or a dangling predecessor is a
// planning error the tech-lead/PM must fix — the scheduler REFUSES to start and
// surfaces it rather than silently breaking a link (#15 step 2, fail-fast).
func Validate(plan *Plan) error {
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
