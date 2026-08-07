package sprintsignal

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

// fakeRunner records every call so a test can assert the exact argv AND the exact
// call count. Both matter: a board read that quietly issues a second call, or one
// whose argv drifts to a default coordinate, degrades into permanent silence —
// which is byte-identical to "there was nothing to report".
type fakeRunner struct {
	calls [][]string
	out   []byte
	err   error
}

func (f *fakeRunner) run(name string, args ...string) ([]byte, error) {
	f.calls = append(f.calls, append([]string{name}, args...))
	return f.out, f.err
}

func issuesJSON(t *testing.T, items ...map[string]any) []byte {
	t.Helper()
	b, err := json.Marshal(items)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func issue(state string, labels ...string) map[string]any {
	ls := []map[string]string{}
	for _, l := range labels {
		ls = append(ls, map[string]string{"name": l})
	}
	return map[string]any{"number": 1, "state": state, "labels": ls}
}

// The read side, pinned. Non-default owner/repo on purpose: a hardcoded
// coordinate would pass against the values this repo happens to use.
func TestScan_ArgvAndCallCount(t *testing.T) {
	f := &fakeRunner{out: issuesJSON(t)}
	if _, err := Scan(f.run, "acme-org", "widget-repo"); err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(f.calls) != 1 {
		t.Fatalf("expected exactly 1 board call, got %d: %v", len(f.calls), f.calls)
	}
	want := []string{
		"gh", "issue", "list",
		"--repo", "acme-org/widget-repo",
		"--state", "all",
		"--limit", "1000",
		"--json", "number,state,labels",
	}
	if !reflect.DeepEqual(f.calls[0], want) {
		t.Errorf("argv drifted\n got: %v\nwant: %v", f.calls[0], want)
	}
}

func TestScan_CountsOpenWorkExcludingCandidates(t *testing.T) {
	f := &fakeRunner{out: issuesJSON(t,
		issue("OPEN"),
		issue("OPEN", "type:bug"),
		issue("CLOSED"),
		// A /request item awaiting the PO's accept is not open work — it is
		// excluded from the ready frontier until accepted.
		issue("OPEN", "candidate", "triage:light"),
	)}
	v, err := Scan(f.run, "o", "r")
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if v.OpenItems != 2 {
		t.Errorf("OpenItems = %d, want 2 (candidate excluded)", v.OpenItems)
	}
	if v.ScannedAt.IsZero() {
		t.Error("ScannedAt must be stamped — Load's age bound depends on it")
	}
}

// The regression the adapter warns about by name: a lexical maximum hands back a
// stale ordinal and the next sprint reuses a number already in use.
func TestScan_HighestSprintComparesAsInteger(t *testing.T) {
	f := &fakeRunner{out: issuesJSON(t,
		issue("CLOSED", "sprint:9"),
		issue("CLOSED", "sprint:10"),
		issue("OPEN", "sprint:2"),
	)}
	v, err := Scan(f.run, "o", "r")
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if v.HighestSprint != 10 {
		t.Errorf("HighestSprint = %d, want 10 (a lexical max would say 9)", v.HighestSprint)
	}
}

// The highest ordinal must be read from every item, not just the open ones: a
// sprint whose work is all closed is still the current sprint until reviewed.
func TestScan_HighestSprintIncludesClosedItems(t *testing.T) {
	f := &fakeRunner{out: issuesJSON(t,
		issue("CLOSED", "sprint:4"),
		issue("OPEN"), // backlog, no sprint label
	)}
	v, err := Scan(f.run, "o", "r")
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if v.HighestSprint != 4 {
		t.Errorf("HighestSprint = %d, want 4 from a fully-closed sprint", v.HighestSprint)
	}
	if v.OpenItems != 1 {
		t.Errorf("OpenItems = %d, want 1", v.OpenItems)
	}
}

func TestScan_NoSprintLabelsAnywhere(t *testing.T) {
	f := &fakeRunner{out: issuesJSON(t, issue("OPEN", "type:feature"))}
	v, err := Scan(f.run, "o", "r")
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if v.HighestSprint != 0 {
		t.Errorf("HighestSprint = %d, want 0 when no item carries a sprint label", v.HighestSprint)
	}
}

// A filled cap means the read may have missed a higher ordinal, so it must fail
// rather than cache a verdict it cannot stand behind.
func TestScan_TruncatedReadIsAnError(t *testing.T) {
	items := make([]map[string]any, 0, ScanLimit)
	for i := 0; i < ScanLimit; i++ {
		items = append(items, issue("CLOSED"))
	}
	f := &fakeRunner{out: issuesJSON(t, items...)}
	if _, err := Scan(f.run, "o", "r"); err == nil {
		t.Fatal("a result filling the cap must be an error, not a verdict")
	}
}

// Every failure of the read must surface as an error — the caller turns that into
// "write no verdict", which turns into silence.
func TestScan_FailuresSurface(t *testing.T) {
	t.Run("command error", func(t *testing.T) {
		f := &fakeRunner{err: fmt.Errorf("gh: not found")}
		if _, err := Scan(f.run, "o", "r"); err == nil {
			t.Fatal("a failing gh must be an error")
		}
	})
	t.Run("unparseable output", func(t *testing.T) {
		f := &fakeRunner{out: []byte("not json")}
		if _, err := Scan(f.run, "o", "r"); err == nil {
			t.Fatal("unparseable output must be an error")
		}
	})
	t.Run("missing coordinates makes no call at all", func(t *testing.T) {
		f := &fakeRunner{out: issuesJSON(t)}
		if _, err := Scan(f.run, "", "r"); err == nil {
			t.Fatal("an empty owner must be an error")
		}
		if len(f.calls) != 0 {
			t.Errorf("expected no board call without coordinates, got %v", f.calls)
		}
	})
}

func TestSprintOrdinal(t *testing.T) {
	cases := []struct {
		label string
		want  int
		ok    bool
	}{
		{"sprint:1", 1, true},
		{"sprint:10", 10, true},
		{"sprint:0", 0, false},   // ordinals are 1-based
		{"sprint:", 0, false},    // no ordinal
		{"sprint:+3", 0, false},  // digits only — Atoi alone would accept this
		{"sprint:-3", 0, false},  // ditto
		{"sprint:2a", 0, false},  //
		{"sprint:foo", 0, false}, // a slug carrier is not the ordinal carrier
		{"sprints:2", 0, false},
		{"type:bug", 0, false},
		{"candidate", 0, false},
	}
	for _, c := range cases {
		got, ok := SprintOrdinal(c.label)
		if ok != c.ok || got != c.want {
			t.Errorf("SprintOrdinal(%q) = (%d,%v), want (%d,%v)", c.label, got, ok, c.want, c.ok)
		}
	}
}

func TestReviewPagePath(t *testing.T) {
	got := ReviewPagePath("/proj", 7)
	want := filepath.Join("/proj", "docs", "sprints", "sprint-7-review.md")
	if got != want {
		t.Errorf("ReviewPagePath = %q, want %q", got, want)
	}
}

func TestNoActiveSprint(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "docs", "sprints"), 0o755); err != nil {
		t.Fatal(err)
	}

	if !(Verdict{HighestSprint: 0}).NoActiveSprint(root) {
		t.Error("no sprint ever opened → no active sprint")
	}
	if (Verdict{HighestSprint: 3}).NoActiveSprint(root) {
		t.Error("sprint 3 with no review page is still current → a sprint IS active")
	}
	if err := os.WriteFile(ReviewPagePath(root, 3), []byte("# review"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !(Verdict{HighestSprint: 3}).NoActiveSprint(root) {
		t.Error("sprint 3 reviewed → it is no longer current")
	}
	// The page for a DIFFERENT sprint must not silence the signal.
	if (Verdict{HighestSprint: 4}).NoActiveSprint(root) {
		t.Error("sprint 4 unreviewed → active, regardless of sprint 3's page")
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "verdict.json")
	now := time.Now().UTC()
	want := Verdict{OpenItems: 12, HighestSprint: 5, ScannedAt: now}
	if err := Save(path, want); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, ok := Load(path, time.Hour, now)
	if !ok {
		t.Fatal("a verdict just saved must load")
	}
	if got.OpenItems != want.OpenItems || got.HighestSprint != want.HighestSprint {
		t.Errorf("round trip = %+v, want %+v", got, want)
	}
}

// Every "this machine has nothing it can stand behind" shape must resolve to the
// same not-ok, so every one of them ends in silence.
func TestLoad_UnusableShapesAllReturnNotOK(t *testing.T) {
	dir := t.TempDir()
	now := time.Now().UTC()

	if _, ok := Load(filepath.Join(dir, "absent.json"), time.Hour, now); ok {
		t.Error("a missing verdict must not load")
	}

	bad := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(bad, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, ok := Load(bad, time.Hour, now); ok {
		t.Error("a malformed verdict must not load")
	}

	noStamp := filepath.Join(dir, "nostamp.json")
	if err := os.WriteFile(noStamp, []byte(`{"openItems":3,"highestSprint":1}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, ok := Load(noStamp, time.Hour, now); ok {
		t.Error("a verdict with no scannedAt has no knowable age — it must not load")
	}

	stale := filepath.Join(dir, "stale.json")
	if err := Save(stale, Verdict{OpenItems: 3, ScannedAt: now.Add(-48 * time.Hour)}); err != nil {
		t.Fatal(err)
	}
	if _, ok := Load(stale, 24*time.Hour, now); ok {
		t.Error("a verdict older than maxAge must not load — a board read that stopped working must stop speaking")
	}
	if _, ok := Load(stale, 72*time.Hour, now); !ok {
		t.Error("the same verdict inside maxAge must load")
	}
}

// The package doc and the code must agree that the sprint label prefix and the
// review page name are the delivery-team's, not invented here.
func TestBindingsMatchTheAdapterContract(t *testing.T) {
	if sprintLabelPrefix != "sprint:" {
		t.Errorf("sprint carrier prefix = %q, adapter §5 says sprint:", sprintLabelPrefix)
	}
	if candidateLabel != "candidate" {
		t.Errorf("candidate label = %q, concept #14 says candidate", candidateLabel)
	}
	if !strings.HasSuffix(ReviewPagePath("/x", 2), filepath.Join("docs", "sprints", "sprint-2-review.md")) {
		t.Error("review page path must be the adapter §9 in-repo docs location")
	}
}
