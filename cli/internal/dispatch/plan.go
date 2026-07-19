// Package dispatch is the deterministic orchestration engine behind
// `atl work dispatch` — the delivery-team's work-unit supervisor.
//
// It holds ZERO LLM context: a process pool of `claude -p` workers, each in its
// own git worktree, scheduled over a dependency DAG and observed only through
// the status.json files the workers write. Durable state lives in Azure; this
// engine never calls Azure (MCP is the workers' surface) and never holds a
// conversation.
//
// The engine consumes two file contracts it does not author:
//
//   - plan.json  — the materialized work-unit DAG a /sprint-start ceremony
//     writes (this file). The DAG was built from Azure over MCP on the ceremony
//     (LLM) side; the Go engine only reads and schedules from it, because it
//     cannot call MCP itself.
//   - status.json — the live per-worker progress a worker writes into its
//     worktree (status.go). The engine polls it for liveness + phase progress.
package dispatch

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Granularity is the work-unit level of a sprint. A sprint is all-PBI OR
// all-task (Resolution #7 — mixed granularity is forbidden), so the DAG never
// spans levels.
type Granularity string

const (
	GranularityPBI  Granularity = "pbi"
	GranularityTask Granularity = "task"
)

// WorkUnit is one schedulable work-item in the plan. Its branch and
// worktree are both derived as delivery/<sprint-slug>/<ID> — there is no
// separate slug field; the ID (the backend's work-item id) makes every branch trace
// to exactly one unit and one PR.
type WorkUnit struct {
	ID           int    `json:"id"`           // the backend's work-item id — the stable identity
	Title        string `json:"title"`        // for logging + PR/branch context
	Predecessors []int  `json:"predecessors"` // work-item ids that must be Done first (dependency links)
	StackRank    int    `json:"stackRank"`    // admission tie-break (lower = higher priority). JSON key: "priority" (preferred, concept #5) OR legacy "stackRank" — see UnmarshalJSON.
}

// UnmarshalJSON accepts either "priority" (the concept-#5 name the ceremony docs
// use) or the legacy "stackRank" (an Azure-ism) as the admission tie-break key, so
// a plan a ceremony emits with EITHER key lands a real value instead of silently
// deserializing to 0 (which would flatten the whole frontier to id-order). "priority"
// wins when both are present.
func (u *WorkUnit) UnmarshalJSON(data []byte) error {
	type alias WorkUnit // shed the method set to avoid infinite recursion
	aux := struct {
		Priority *int `json:"priority"`
		*alias
	}{alias: (*alias)(u)}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if aux.Priority != nil {
		u.StackRank = *aux.Priority
	}
	return nil
}

// Plan is the handoff artifact: the materialized DAG a /sprint-start ceremony
// writes to <project>/.delivery/plan.json and `atl work dispatch` reads. It is
// pure already-resolved data.
type Plan struct {
	SprintSlug  string      `json:"sprintSlug"`  // FS-safe sprint id; the {slug} in delivery/{slug}/{id}
	Granularity Granularity `json:"granularity"` // pbi | task (all units one level)
	Units       []WorkUnit  `json:"units"`
}

// PlanPath returns the canonical plan-artifact location for a project:
// <projectRoot>/.delivery/plan.json — the file a /sprint-start ceremony writes
// and `atl work dispatch` reads.
func PlanPath(projectRoot string) string {
	return filepath.Join(projectRoot, ".delivery", "plan.json")
}

// Load reads and parses the plan at path.
func Load(path string) (*Plan, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var p Plan
	if err := json.Unmarshal(b, &p); err != nil {
		return nil, fmt.Errorf("parse plan %s: %w", path, err)
	}
	return &p, nil
}
