package dispatch

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// BlockedReport is the supervisor's durable, LLM-free record of a work-unit the
// recovery ladder (#12/#5) gave up on. The deterministic engine deliberately has
// NO Azure MCP surface, so it cannot set the work-item to Blocked itself; instead
// it writes this report to <project>/.delivery/blocked/<id>.json and surfaces it,
// and a later ceremony skill (which owns the Azure surface) reflects it to the
// work-item's Blocked state + diagnostic comment. Same file-contract pattern as
// plan.json (skill → engine) and status.json (worker → engine): the engine
// produces durable data at the boundary; the LLM layer syncs it to Azure.
type BlockedReport struct {
	ID           int       `json:"id"`
	Branch       string    `json:"branch"`               // the (possibly quarantined) branch preserved for diagnosis
	WorktreePath string    `json:"worktreePath"`         // where the preserved worktree sits; empty when the clean worktree was reclaimed
	Reason       string    `json:"reason"`               // stall / crash / worker-reported — human-readable
	Phase        string    `json:"phase"`                // last observed worker phase
	LastSummary  string    `json:"lastSummary"`          // status.json lastOutputSummary at block time
	StderrTail   string    `json:"stderrTail,omitempty"` // last stderr for a hard-crash diagnostic
	Preserved    bool      `json:"preserved"`            // true = branch/worktree kept (unmerged/dirty); false = reclaimed clean
	BlockedAt    time.Time `json:"blockedAt"`
}

// BlockedDir is the canonical directory of per-unit blocked reports for a
// project: <projectRoot>/.delivery/blocked/.
func BlockedDir(projectRoot string) string {
	return filepath.Join(projectRoot, ".delivery", "blocked")
}

// WriteBlockedReport atomically writes r to <project>/.delivery/blocked/<id>.json
// (tmp + rename, the manifest.go idiom). One file per unit so a re-block
// overwrites its own report and a ceremony skill can drain the directory.
func WriteBlockedReport(projectRoot string, r *BlockedReport) error {
	dir := BlockedDir(projectRoot)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	path := filepath.Join(dir, strconv.Itoa(r.ID)+".json")
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(b, '\n'), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
