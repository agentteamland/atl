package dispatch

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWriteBlockedReportRoundTrip(t *testing.T) {
	root := t.TempDir()
	r := &BlockedReport{
		ID:           4821,
		Branch:       "delivery/s14/4821-blocked1",
		WorktreePath: filepath.Join(root, ".delivery", "worktrees", ".quarantine", "s14", "4821-blocked1"),
		Reason:       "stalled: phase-stall-breach (retry also failed)",
		Phase:        "self-test",
		LastSummary:  "re-running the flaky suite",
		StderrTail:   "panic: nil map",
		Preserved:    true,
		BlockedAt:    time.Unix(1_700_000_000, 0).UTC(),
	}
	if err := WriteBlockedReport(root, r); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(BlockedDir(root), "4821.json")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("report not at the canonical per-id path: %v", err)
	}
	var got BlockedReport
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("report is not valid JSON: %v", err)
	}
	if got.ID != r.ID || got.Branch != r.Branch || got.Reason != r.Reason || !got.Preserved {
		t.Errorf("round-trip lost fields: %+v", got)
	}
	// No leftover temp file from the atomic write.
	if _, err := os.Stat(path + ".tmp"); !os.IsNotExist(err) {
		t.Error("atomic write left a .tmp file behind")
	}
}

func TestWriteBlockedReportOverwritesSameUnit(t *testing.T) {
	root := t.TempDir()
	if err := WriteBlockedReport(root, &BlockedReport{ID: 7, Reason: "first"}); err != nil {
		t.Fatal(err)
	}
	if err := WriteBlockedReport(root, &BlockedReport{ID: 7, Reason: "second"}); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(BlockedDir(root), "7.json"))
	if err != nil {
		t.Fatal(err)
	}
	var got BlockedReport
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if got.Reason != "second" {
		t.Errorf("a re-block should overwrite its own report; reason = %q", got.Reason)
	}
}
