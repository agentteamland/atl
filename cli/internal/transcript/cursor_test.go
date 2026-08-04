package transcript

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeFlow writes a transcript of user/assistant prose turns and returns its path.
func writeFlow(t *testing.T, dir, name string, turns ...[2]string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	var sb strings.Builder
	for _, tn := range turns {
		sb.WriteString(`{"message":{"role":"` + tn[0] + `","content":"` + tn[1] + `"}}` + "\n")
	}
	if err := os.WriteFile(p, []byte(sb.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func appendFlow(t *testing.T, path string, turns ...[2]string) {
	t.Helper()
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	for _, tn := range turns {
		if _, err := f.WriteString(`{"message":{"role":"` + tn[0] + `","content":"` + tn[1] + `"}}` + "\n"); err != nil {
			t.Fatal(err)
		}
	}
}

func texts(turns []Turn) []string {
	out := make([]string, len(turns))
	for i, t := range turns {
		out[i] = t.Text
	}
	return out
}

// The whole point of the cursor: what the first read did not see, the second one
// does — and it sees it exactly once.
func TestExtractFlowFromResumesAtTheBoundary(t *testing.T) {
	dir := t.TempDir()
	p := writeFlow(t, dir, "s.jsonl", [2]string{"user", "one"}, [2]string{"assistant", "two"})

	first, err := ExtractFlowFrom(p, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if got := texts(first.Turns); len(got) != 2 || got[0] != "one" || got[1] != "two" {
		t.Fatalf("first read = %v, want [one two]", got)
	}
	if first.Pending {
		t.Fatal("unbudgeted read reported pending")
	}

	appendFlow(t, p, [2]string{"user", "three"})
	second, err := ExtractFlowFrom(p, first.Next, 0)
	if err != nil {
		t.Fatal(err)
	}
	if got := texts(second.Turns); len(got) != 1 || got[0] != "three" {
		t.Fatalf("resumed read = %v, want [three]", got)
	}

	// And a third read with nothing appended yields nothing rather than repeating.
	third, err := ExtractFlowFrom(p, second.Next, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(third.Turns) != 0 {
		t.Fatalf("read past the end = %v, want none", texts(third.Turns))
	}
}

// A budgeted read must stop cleanly: the caller resumes at the first turn it did
// NOT receive, so several bounded sweeps add up to full coverage.
func TestExtractFlowFromBudgetHandsOffCleanly(t *testing.T) {
	dir := t.TempDir()
	p := writeFlow(t, dir, "s.jsonl",
		[2]string{"user", "aaaa"}, [2]string{"assistant", "bbbb"}, [2]string{"user", "cccc"})

	var got []string
	off := int64(0)
	for i := 0; i < 5; i++ {
		fl, err := ExtractFlowFrom(p, off, 4) // room for one 4-byte turn at a time
		if err != nil {
			t.Fatal(err)
		}
		got = append(got, texts(fl.Turns)...)
		off = fl.Next
		if !fl.Pending {
			break
		}
	}
	if strings.Join(got, ",") != "aaaa,bbbb,cccc" {
		t.Fatalf("budgeted sweeps = %v, want each turn exactly once in order", got)
	}
}

// One turn larger than the whole budget must still come back, or the cursor
// wedges on that record and the channel stops mining forever.
func TestExtractFlowFromOversizedTurnStillAdvances(t *testing.T) {
	dir := t.TempDir()
	big := strings.Repeat("x", 500)
	p := writeFlow(t, dir, "s.jsonl", [2]string{"user", big}, [2]string{"assistant", "next"})

	fl, err := ExtractFlowFrom(p, 0, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(fl.Turns) != 1 || fl.Turns[0].Text != big {
		t.Fatalf("oversized turn not returned: %d turn(s)", len(fl.Turns))
	}
	if !fl.Pending || fl.Next == 0 {
		t.Fatalf("oversized turn did not advance: pending=%v next=%d", fl.Pending, fl.Next)
	}
	rest, err := ExtractFlowFrom(p, fl.Next, 10)
	if err != nil {
		t.Fatal(err)
	}
	if got := texts(rest.Turns); len(got) != 1 || got[0] != "next" {
		t.Fatalf("resume after oversized turn = %v, want [next]", got)
	}
}

// A half-written trailing record is emitted (it may be a whole turn) but must
// never be counted as consumed — the writer may still be extending it.
func TestExtractFlowFromDoesNotConsumeAnUnterminatedRecord(t *testing.T) {
	dir := t.TempDir()
	p := writeFlow(t, dir, "s.jsonl", [2]string{"user", "done"})
	f, err := os.OpenFile(p, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	partial := `{"message":{"role":"assistant","content":"still writing"}}` // no newline
	if _, err := f.WriteString(partial); err != nil {
		t.Fatal(err)
	}
	f.Close()

	fl, err := ExtractFlowFrom(p, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if got := texts(fl.Turns); len(got) != 2 {
		t.Fatalf("trailing record not emitted: %v", got)
	}
	info, err := os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	if fl.Next != info.Size()-int64(len(partial)) {
		t.Fatalf("Next = %d, want the boundary before the unterminated record (%d)", fl.Next, info.Size()-int64(len(partial)))
	}
}

// A stretch of records that yields no prose at all (tool traffic, harness
// injections) must still move the cursor, or the sweep stalls on it.
func TestExtractFlowFromAdvancesOverProselessRecords(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "s.jsonl")
	body := `{"message":{"role":"user","content":""}}` + "\n" +
		`{"isMeta":true,"message":{"role":"user","content":"skill dump"}}` + "\n" +
		"\n" // a blank line
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	fl, err := ExtractFlowFrom(p, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(fl.Turns) != 0 {
		t.Fatalf("expected no prose, got %v", texts(fl.Turns))
	}
	if fl.Next != int64(len(body)) {
		t.Fatalf("Next = %d, want %d — a proseless stretch must not stall the cursor", fl.Next, len(body))
	}
}

// CRLF must not desync the byte accounting, or every resumed offset after the
// first line lands mid-record.
func TestExtractFlowFromHandlesCRLF(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "s.jsonl")
	body := `{"message":{"role":"user","content":"one"}}` + "\r\n" +
		`{"message":{"role":"assistant","content":"two"}}` + "\r\n"
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	first, err := ExtractFlowFrom(p, 0, 3)
	if err != nil {
		t.Fatal(err)
	}
	if got := texts(first.Turns); len(got) != 1 || got[0] != "one" {
		t.Fatalf("first = %v, want [one]", got)
	}
	second, err := ExtractFlowFrom(p, first.Next, 0)
	if err != nil {
		t.Fatal(err)
	}
	if got := texts(second.Turns); len(got) != 1 || got[0] != "two" {
		t.Fatalf("resumed across CRLF = %v, want [two]", got)
	}
}

func TestCursorRoundTripsPerChannel(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	project := "/some/project"

	c, err := LoadCursor(project)
	if err != nil {
		t.Fatal(err)
	}
	if c.Known("learning") {
		t.Fatal("a fresh cursor reported a known channel")
	}
	c.SetOffset("learning", "a.jsonl", 100)
	c.SetOffset("profile-fact", "a.jsonl", 40)
	if err := c.Save(project); err != nil {
		t.Fatal(err)
	}

	got, err := LoadCursor(project)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Known("learning") || !got.Known("profile-fact") {
		t.Fatal("channels did not survive the round trip")
	}
	// The channels must not share a position: one drain advancing its own cursor
	// is exactly what must NOT consume the other's unmined window.
	if off, _ := got.Offset("learning", "a.jsonl"); off != 100 {
		t.Fatalf("learning offset = %d, want 100", off)
	}
	if off, _ := got.Offset("profile-fact", "a.jsonl"); off != 40 {
		t.Fatalf("profile-fact offset = %d, want 40", off)
	}
	if _, ok := got.Offset("learning", "unknown.jsonl"); ok {
		t.Fatal("an unrecorded transcript reported an offset")
	}
}

func TestCursorPruneDropsVanishedTranscripts(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	c, err := LoadCursor("/p")
	if err != nil {
		t.Fatal(err)
	}
	c.SetOffset("learning", "gone.jsonl", 1)
	c.SetOffset("learning", "here.jsonl", 2)
	c.Prune("learning", map[string]bool{"here.jsonl": true})
	if _, ok := c.Offset("learning", "gone.jsonl"); ok {
		t.Fatal("a deleted transcript stayed in the cursor")
	}
	if _, ok := c.Offset("learning", "here.jsonl"); !ok {
		t.Fatal("prune dropped a live transcript")
	}
}

// An unreadable cursor must degrade to "never mined" (re-read, safe) rather than
// fail the drain that depends on it.
func TestLoadCursorTreatsCorruptStateAsEmpty(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	p, err := CursorPath("/p")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	c, err := LoadCursor("/p")
	if err != nil {
		t.Fatalf("corrupt cursor returned an error: %v", err)
	}
	if c.Known("learning") {
		t.Fatal("corrupt cursor reported a known channel")
	}
}
