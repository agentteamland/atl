package transcript

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestSlugForPath(t *testing.T) {
	cases := []struct{ path, want string }{
		// dot-free path — leading slash + separators become hyphens
		{"/w/projects/myapp", "-w-projects-myapp"},
		// a dot in a path component (.claude) must slug to a hyphen too — this is the
		// worktree case that silently broke capture. Synthetic path; mirrors a real
		// on-disk Claude Code worktree slug (.../p/.claude/worktrees/y -> ...p--claude-worktrees-y).
		{
			"/w/proj/.claude/worktrees/dazzling-morse-b2e4ae",
			"-w-proj--claude-worktrees-dazzling-morse-b2e4ae",
		},
		// every other non-alphanumeric also folds to a hyphen
		{"/a/b.c/d_e", "-a-b-c-d-e"},
	}
	for _, c := range cases {
		if got := SlugForPath(c.path); got != c.want {
			t.Errorf("SlugForPath(%q) = %q, want %q", c.path, got, c.want)
		}
	}
}

func TestExtractText(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "t.jsonl")
	lines := `{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"Decided. <!-- learning: A -->"},{"type":"tool_use","name":"x"}]}}
{"type":"user","message":{"role":"user","content":[{"type":"text","text":"<!-- learning: from-user-ignored -->"}]}}
{"type":"assistant","message":{"role":"assistant","content":"string shape <!-- profile-fact: B -->"}}
not even json
`
	if err := os.WriteFile(path, []byte(lines), 0o600); err != nil {
		t.Fatal(err)
	}
	text, skipped, err := ExtractText(path)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if skipped != 0 {
		t.Errorf("skipped = %d, want 0 (no over-long records here)", skipped)
	}
	if !strings.Contains(text, "<!-- learning: A -->") {
		t.Error("missing assistant array-content marker")
	}
	if !strings.Contains(text, "<!-- profile-fact: B -->") {
		t.Error("missing assistant string-content marker")
	}
	if strings.Contains(text, "from-user-ignored") {
		t.Error("user-message text must be ignored")
	}
}

func TestExtractFlow(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "t.jsonl")
	lines := `{"type":"user","message":{"role":"user","content":[{"type":"text","text":"do the auth thing"}]}}
{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"on it"},{"type":"tool_use","name":"Edit"}]}}
{"type":"user","message":{"role":"user","content":[{"type":"tool_result","content":"ok"}]}}
{"type":"user","message":{"role":"user","content":"no, use refresh tokens not sessions"}}
{"type":"assistant","message":{"role":"assistant","content":"fixed"}}
not even json
`
	if err := os.WriteFile(path, []byte(lines), 0o600); err != nil {
		t.Fatal(err)
	}
	turns, skipped, err := ExtractFlow(path)
	if err != nil {
		t.Fatalf("flow: %v", err)
	}
	if skipped != 0 {
		t.Errorf("skipped = %d, want 0 (no over-long records here)", skipped)
	}
	// Expected: user "do the auth thing", assistant "on it", user "no, use…",
	// assistant "fixed". The tool_result-only user message yields no text → dropped.
	if len(turns) != 4 {
		t.Fatalf("want 4 prose turns, got %d: %+v", len(turns), turns)
	}
	if turns[0].Role != "user" || turns[0].Text != "do the auth thing" {
		t.Errorf("turn0: %+v", turns[0])
	}
	if turns[1].Role != "assistant" || turns[1].Text != "on it" {
		t.Errorf("turn1 (tool_use must be dropped): %+v", turns[1])
	}
	if turns[2].Role != "user" || !strings.Contains(turns[2].Text, "refresh tokens") {
		t.Errorf("turn2 (the correction): %+v", turns[2])
	}
	for _, tn := range turns {
		if strings.Contains(tn.Text, "ok") && tn.Text == "ok" {
			t.Error("tool_result content must not become a turn")
		}
	}
}

func TestFindModTimeFilterAndOrder(t *testing.T) {
	dir := t.TempDir()
	old := filepath.Join(dir, "old.jsonl")
	recent := filepath.Join(dir, "recent.jsonl")
	_ = os.WriteFile(old, []byte("{}"), 0o600)
	_ = os.WriteFile(recent, []byte("{}"), 0o600)
	t0 := time.Now().Add(-2 * time.Hour)
	t1 := time.Now().Add(-1 * time.Minute)
	_ = os.Chtimes(old, t0, t0)
	_ = os.Chtimes(recent, t1, t1)

	// since 1h ago → only recent
	files, err := Find(dir, time.Now().Add(-1*time.Hour))
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if len(files) != 1 || filepath.Base(files[0].Path) != "recent.jsonl" {
		t.Fatalf("modtime filter: got %+v", files)
	}

	// zero since → both, oldest first
	all, _ := Find(dir, time.Time{})
	if len(all) != 2 || filepath.Base(all[0].Path) != "old.jsonl" {
		t.Fatalf("ordering: got %+v", all)
	}
}

func TestFindMissingDir(t *testing.T) {
	files, err := Find(filepath.Join(t.TempDir(), "nope"), time.Time{})
	if err != nil {
		t.Fatalf("missing dir should not error: %v", err)
	}
	if files != nil {
		t.Fatalf("want nil, got %+v", files)
	}
}

func TestFindIgnoresNonJsonl(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "a.jsonl"), []byte("{}"), 0o600)
	_ = os.WriteFile(filepath.Join(dir, "b.txt"), []byte("x"), 0o600)
	files, _ := Find(dir, time.Time{})
	if len(files) != 1 {
		t.Fatalf("want 1 jsonl, got %d", len(files))
	}
}

// dryLine builds one JSONL transcript record for DryStretch tests.
func dryLine(role, text string) string {
	return `{"message":{"role":"` + role + `","content":[{"type":"text","text":` + strconv.Quote(text) + `}]}}` + "\n"
}

func TestDryStretch(t *testing.T) {
	isMarker := func(s string) bool { return strings.Contains(s, "<!-- learning:") }
	dir := t.TempDir()

	write := func(name, body string) string {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		return p
	}

	// A marker-bearing turn resets the stretch: only what follows it counts.
	p := write("a.jsonl",
		dryLine("user", strings.Repeat("x", 500))+
			dryLine("assistant", "reply one")+
			dryLine("assistant", "tool-split continuation")+ // collapses into the same logical turn
			dryLine("user", strings.Repeat("y", 700))+
			dryLine("assistant", "noted <!-- learning: something -->")+
			dryLine("user", strings.Repeat("z", 300))+
			dryLine("assistant", "dry reply A")+
			dryLine("user", strings.Repeat("w", 200))+
			dryLine("assistant", "dry reply B"))
	st, err := DryStretch(p, isMarker)
	if err != nil {
		t.Fatal(err)
	}
	if st.Turns != 2 {
		t.Errorf("Turns = %d, want 2 (only turns after the marker)", st.Turns)
	}
	if st.Chars != 500 {
		t.Errorf("Chars = %d, want 500 (300+200 after the marker)", st.Chars)
	}
	if st.Key != "a.jsonl:1" {
		t.Errorf("Key = %q, want a.jsonl:1", st.Key)
	}

	// No marker at all: everything counts, ordinal 0.
	p2 := write("b.jsonl",
		dryLine("user", strings.Repeat("q", 1200))+
			dryLine("assistant", "one")+
			dryLine("user", "more")+
			dryLine("assistant", "two"))
	st2, err := DryStretch(p2, isMarker)
	if err != nil {
		t.Fatal(err)
	}
	if st2.Turns != 2 || st2.Chars != 1204 {
		t.Errorf("no-marker stretch = %+v, want Turns=2 Chars=1204", st2)
	}
	if st2.Key != "b.jsonl:0" {
		t.Errorf("Key = %q, want b.jsonl:0", st2.Key)
	}

	// Consecutive assistant records collapse: 3 records, 1 logical turn.
	p3 := write("c.jsonl",
		dryLine("user", "hi")+
			dryLine("assistant", "part 1")+
			dryLine("assistant", "part 2")+
			dryLine("assistant", "part 3"))
	st3, err := DryStretch(p3, isMarker)
	if err != nil {
		t.Fatal(err)
	}
	if st3.Turns != 1 {
		t.Errorf("collapsed Turns = %d, want 1", st3.Turns)
	}

	// A marker in a tool-split continuation still ends the stretch (the collapse
	// happens BEFORE marker detection, so the logical turn carries it).
	p4 := write("d.jsonl",
		dryLine("user", "start")+
			dryLine("assistant", "plain part")+
			dryLine("assistant", "tail part <!-- learning: split -->")+
			dryLine("user", "after")+
			dryLine("assistant", "dry"))
	st4, err := DryStretch(p4, isMarker)
	if err != nil {
		t.Fatal(err)
	}
	if st4.Turns != 1 || st4.Chars != 5 || st4.Key != "d.jsonl:1" {
		t.Errorf("split-marker stretch = %+v, want Turns=1 Chars=5 Key=d.jsonl:1", st4)
	}

	// Machine-injected user-role content must not count as user-authored chars:
	// isMeta records (a skill's SKILL.md dump) are dropped from the flow, and
	// tag-opening envelopes (task notifications, command wrappers) are skipped
	// by the counter. Only the two genuinely typed 4-rune messages count — and
	// as RUNES, not UTF-8 bytes (multi-byte input must not inflate the gate).
	p5 := write("e.jsonl",
		`{"isMeta":true,"message":{"role":"user","content":[{"type":"text","text":"`+strings.Repeat("m", 5000)+`"}]}}`+"\n"+
			dryLine("user", "<task-notification>"+strings.Repeat("n", 3000)+"</task-notification>")+
			dryLine("assistant", "reply one")+
			dryLine("user", "café")+ // 4 runes, 5 bytes
			dryLine("assistant", "reply two")+
			dryLine("user", "déjà")) // 4 runes, 6 bytes
	st5, err := DryStretch(p5, isMarker)
	if err != nil {
		t.Fatal(err)
	}
	if st5.Turns != 2 {
		t.Errorf("machine-noise Turns = %d, want 2", st5.Turns)
	}
	if st5.Chars != 8 {
		t.Errorf("machine-noise Chars = %d, want 8 (runes of the two typed messages only)", st5.Chars)
	}
}

// compactLine renders a compaction-summary record: user-role, machine-authored,
// flagged with isCompactSummary.
func compactLine(text string) string {
	return `{"isCompactSummary":true,"message":{"role":"user","content":[{"type":"text","text":` + strconv.Quote(text) + `}]}}` + "\n"
}

// TestDryStretchSkipsMachineUserTurns is the regression for the watchdog's
// char gate. `isMeta` and the `<`-prefix heuristic already covered two carriers
// of machine-authored user-role content; two more slipped through and were
// large enough to satisfy the 1000-char threshold on their own:
//
//   - the compaction recap the harness injects after compacting a long session
//     (13,995-17,966 chars observed in real transcripts), and
//   - the `[Request interrupted by user]` cancellation notice.
//
// Measured on one real project: 127,362 machine chars counted as user input
// against 140,900 genuinely typed — so after any compaction the gate was a
// formality and the watchdog degraded to turns-only.
func TestDryStretchSkipsMachineUserTurns(t *testing.T) {
	isMarker := func(s string) bool { return strings.Contains(s, "<!-- learning:") }
	dir := t.TempDir()
	write := func(name, body string) string {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		return p
	}

	// A dry stretch carrying 16k chars of machine-authored "user input" and only
	// 40 chars the user actually typed. Before the fix the gate saw 16,040 and
	// sailed past its 1000-char threshold on the recap alone.
	p := write("machine.jsonl",
		dryLine("assistant", "noted <!-- learning: something -->")+
			dryLine("user", strings.Repeat("t", 40))+
			dryLine("assistant", "dry reply A")+
			compactLine(strings.Repeat("m", 16000))+
			dryLine("user", "[Request interrupted by user]")+
			dryLine("assistant", "dry reply B"))
	st, err := DryStretch(p, isMarker)
	if err != nil {
		t.Fatal(err)
	}
	if st.Chars != 40 {
		t.Errorf("Chars = %d, want 40 — a 16k compaction recap and an interrupt notice are not user input", st.Chars)
	}
	// Turns = 1: dropping the machine records leaves the two assistant replies
	// adjacent, so the logical-turn collapse merges them. Accepted side effect —
	// a skipped record is removed, not turned into a boundary — and it errs
	// toward SILENCE (fewer counted turns is harder to trip), which is the
	// watchdog's documented failure direction.
	if st.Turns != 1 {
		t.Errorf("Turns = %d, want 1 (the assistant replies collapse once the machine records between them are dropped)", st.Turns)
	}

	// Genuine input alongside the machine carriers is still counted, so the fix
	// suppresses noise without blinding the gate.
	p2 := write("mixed.jsonl",
		dryLine("assistant", "noted <!-- learning: something -->")+
			compactLine(strings.Repeat("m", 16000))+
			dryLine("user", strings.Repeat("t", 640))+
			dryLine("assistant", "dry reply"))
	st2, err := DryStretch(p2, isMarker)
	if err != nil {
		t.Fatal(err)
	}
	if st2.Chars != 640 {
		t.Errorf("Chars = %d, want 640 (only the typed turn)", st2.Chars)
	}
}

// TestSkipsOverLongRecord is the regression for the whole-file abort. A record
// past the size cap put bufio.Scanner into a permanent error state, and both
// readers turned that into a failed read of the ENTIRE file — so one pathological
// line (a pasted blob, a base64 payload, a tool result echoing a big file) cost
// the whole session: the mining input, and via DryStretch the capture watchdog,
// which swallows a read error silently. The record is skipped and counted now.
func TestSkipsOverLongRecord(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "t.jsonl")
	overLong := `{"message":{"role":"assistant","content":"OVERLONG` +
		strings.Repeat("x", maxRecordBytes) + `"}}` + "\n"
	body := dryLine("user", "before the blob") +
		dryLine("assistant", "reply one <!-- learning: A -->") +
		overLong +
		dryLine("user", "after the blob") +
		dryLine("assistant", "reply two <!-- learning: B -->")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}

	text, skipped, err := ExtractText(path)
	if err != nil {
		t.Fatalf("one over-long record must not fail the file: %v", err)
	}
	if skipped != 1 {
		t.Errorf("ExtractText skipped = %d, want 1", skipped)
	}
	if !strings.Contains(text, "<!-- learning: A -->") || !strings.Contains(text, "<!-- learning: B -->") {
		t.Error("markers on BOTH sides of the over-long record must survive")
	}
	if strings.Contains(text, "OVERLONG") {
		t.Error("the over-long record must be skipped, not read into the text")
	}

	turns, skipped, err := ExtractFlow(path)
	if err != nil {
		t.Fatalf("one over-long record must not fail the flow: %v", err)
	}
	if skipped != 1 {
		t.Errorf("ExtractFlow skipped = %d, want 1", skipped)
	}
	if len(turns) != 4 {
		t.Fatalf("want the 4 prose turns around the blob, got %d: %+v", len(turns), turns)
	}
	if turns[3].Text != "reply two <!-- learning: B -->" {
		t.Errorf("turns after the over-long record must survive: %+v", turns[3])
	}

	// The watchdog's input is the same read, so it keeps measuring the session
	// instead of going quiet: both marker-bearing turns are seen (ordinal 2).
	st, err := DryStretch(path, func(s string) bool { return strings.Contains(s, "<!-- learning:") })
	if err != nil {
		t.Fatalf("watchdog must still measure the session: %v", err)
	}
	if st.Key != "t.jsonl:2" {
		t.Errorf("Key = %q, want t.jsonl:2", st.Key)
	}

	// The EOF flush has to count as well as emit: an over-long record that ENDS
	// the file with no trailing newline, on a read-window boundary, never gets a
	// terminator and arrives only as a bare EOF. Skipping it there keeps the count
	// honest — dropping it would be a loss that not even `skipped` records.
	u := filepath.Join(dir, "u.jsonl")
	blob := strings.Repeat("x", 129*readBufBytes) // over the cap AND window-aligned
	if err := os.WriteFile(u, []byte(dryLine("user", "before the blob")+blob), 0o600); err != nil {
		t.Fatal(err)
	}
	turns, skipped, err = ExtractFlow(u)
	if err != nil || skipped != 1 || len(turns) != 1 {
		t.Errorf("unterminated over-long tail: turns=%d skipped=%d err=%v, want 1 1 <nil>", len(turns), skipped, err)
	}
}

// TestFinalRecordWithoutTrailingNewline pins the EOF contract the bufio.Scanner
// replacement has to preserve. A transcript's last record can arrive without its
// newline — the file is still being appended to, or a write was cut short — and
// Scanner emitted that record anyway. bufio.Reader.ReadLine does not hand it over
// the same way: a record whose length lands exactly on the read-window boundary
// ends on an isPrefix chunk with no terminator, and the NEXT call returns only
// io.EOF, so a reader that returns there drops the buffered record with no error
// and no skipped count. That is the same silent loss this whole fix exists to
// remove, so the tail has to be flushed at EOF.
func TestFinalRecordWithoutTrailingNewline(t *testing.T) {
	// sized builds a valid record of exactly n bytes (no newline), so the last
	// one can be placed on the read-window boundary.
	sized := func(role string, n int) string {
		base := len(strings.TrimSuffix(dryLine(role, ""), "\n"))
		if n < base {
			t.Fatalf("cannot build a %d-byte record (minimum %d)", n, base)
		}
		rec := strings.TrimSuffix(dryLine(role, strings.Repeat("x", n-base)), "\n")
		if len(rec) != n {
			t.Fatalf("built a %d-byte record, want %d", len(rec), n)
		}
		return rec
	}

	for _, tc := range []struct {
		name string
		last string
	}{
		{"short", strings.TrimSuffix(dryLine("assistant", "last word <!-- learning: Z -->"), "\n")},
		{"exactly one read window", sized("assistant", readBufBytes)},
		{"exactly two read windows", sized("assistant", 2*readBufBytes)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "t.jsonl")
			body := dryLine("user", "first") + tc.last // deliberately no trailing newline
			if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
				t.Fatal(err)
			}
			turns, skipped, err := ExtractFlow(path)
			if err != nil {
				t.Fatalf("flow: %v", err)
			}
			if skipped != 0 {
				t.Errorf("skipped = %d, want 0 (nothing here is over-long)", skipped)
			}
			if len(turns) != 2 {
				t.Fatalf("got %d turns, want 2 — the newline-less final record was dropped", len(turns))
			}
			text, _, err := ExtractText(path)
			if err != nil {
				t.Fatalf("text: %v", err)
			}
			if strings.TrimSpace(text) == "" {
				t.Error("the final assistant record carried no text into ExtractText")
			}
		})
	}
}

// TestExtractFlowSkipsMachineUserTurns keeps the mining surface consistent with
// the watchdog: a compaction recap is not conversation to mine either.
func TestExtractFlowSkipsMachineUserTurns(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "t.jsonl")
	body := dryLine("user", "real question") +
		compactLine("This session is being continued from a previous conversation...") +
		dryLine("user", "[Request interrupted by user]") +
		dryLine("assistant", "real answer")
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	turns, _, err := ExtractFlow(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(turns) != 2 {
		t.Fatalf("got %d turns, want 2 (the two machine records must be skipped): %+v", len(turns), turns)
	}
	if turns[0].Text != "real question" || turns[1].Text != "real answer" {
		t.Errorf("wrong turns survived: %+v", turns)
	}
}
