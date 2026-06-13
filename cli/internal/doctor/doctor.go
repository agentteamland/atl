// Package doctor runs platform health checks and, where it safely can,
// self-heals.
//
// In v2 the same checks run automatically every session and repair what they
// can — but "repair" is bounded by the CLI/Skill split: deterministic fixes
// (re-bind a hook, retry a fan-out) belong here, while processing a queue item
// into a knowledge base needs an LLM and is a skill's job. For the latter the
// doctor *signals* rather than fixes. `atl doctor` is the on-demand surface
// for the same checks. (Real self-heals land as their dependencies arrive —
// hook re-bind with the hooks phase, fan-out retry with fan-out.)
package doctor

import (
	"fmt"
	"time"

	"github.com/agentteamland/atl/cli/internal/queue"
)

// Status is a check outcome, ordered by severity.
type Status int

const (
	OK Status = iota
	Warn
	Fail
)

func (s Status) String() string {
	switch s {
	case OK:
		return "OK"
	case Warn:
		return "WARN"
	default:
		return "FAIL"
	}
}

// Result is the outcome of one check.
type Result struct {
	Name   string
	Status Status
	Detail string
	Healed bool // a deterministic self-heal was applied during the check
}

// Check inspects (and may self-heal) one aspect of platform health.
type Check func() Result

// Run executes checks in order.
func Run(checks []Check) []Result {
	out := make([]Result, 0, len(checks))
	for _, c := range checks {
		out = append(out, c())
	}
	return out
}

// Worst returns the most severe status across results (OK if empty).
func Worst(results []Result) Status {
	worst := OK
	for _, r := range results {
		if r.Status > worst {
			worst = r.Status
		}
	}
	return worst
}

// Tunables for the queue checks. Constants for now; config-driven later.
const (
	BacklogWarn    = 50              // pending items above this => drain isn't keeping up
	CursorStaleAge = 24 * time.Hour // cursor older than this (with items) => ticks not running
)

// QueueChecks returns the learning-queue health checks for project, evaluated
// against now.
func QueueChecks(store *queue.Store, project string, now time.Time) []Check {
	return []Check{
		func() Result { return backlogCheck(store, project) },
		func() Result { return cursorCheck(store, project, now) },
	}
}

func pendingTotal(store *queue.Store, project string) (int, error) {
	counts, err := store.Counts(project)
	if err != nil {
		return 0, err
	}
	total := 0
	for _, n := range counts {
		total += n
	}
	return total, nil
}

func backlogCheck(store *queue.Store, project string) Result {
	total, err := pendingTotal(store, project)
	if err != nil {
		return Result{Name: "queue-backlog", Status: Fail, Detail: "cannot read queue: " + err.Error()}
	}
	switch {
	case total == 0:
		return Result{Name: "queue-backlog", Status: OK, Detail: "queue empty"}
	case total > BacklogWarn:
		return Result{Name: "queue-backlog", Status: Warn,
			Detail: fmt.Sprintf("%d pending items — a drain skill should process them", total)}
	default:
		return Result{Name: "queue-backlog", Status: OK, Detail: fmt.Sprintf("%d pending item(s)", total)}
	}
}

func cursorCheck(store *queue.Store, project string, now time.Time) Result {
	pending, err := pendingTotal(store, project)
	if err != nil {
		return Result{Name: "tick-freshness", Status: Fail, Detail: "cannot read queue: " + err.Error()}
	}
	cur, err := store.Cursor(project)
	if err != nil {
		return Result{Name: "tick-freshness", Status: Fail, Detail: "cannot read cursor: " + err.Error()}
	}
	if cur.IsZero() {
		if pending > 0 {
			return Result{Name: "tick-freshness", Status: Warn,
				Detail: "never ticked but items are queued — is tick wired?"}
		}
		return Result{Name: "tick-freshness", Status: OK, Detail: "no ticks yet (nothing queued)"}
	}
	age := now.Sub(cur)
	if age > CursorStaleAge && pending > 0 {
		return Result{Name: "tick-freshness", Status: Warn,
			Detail: fmt.Sprintf("last tick %s ago with items pending — ticks may not be running", age.Round(time.Hour))}
	}
	return Result{Name: "tick-freshness", Status: OK, Detail: fmt.Sprintf("last tick %s ago", age.Round(time.Second))}
}
