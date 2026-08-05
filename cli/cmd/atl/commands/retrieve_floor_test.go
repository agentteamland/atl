package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/agentteamland/atl/cli/internal/retrieve"
	"github.com/spf13/cobra"
)

// hookOut runs the hook body against a prompt in a throwaway project + HOME and
// returns what it printed. No index exists, so nothing is ever injected — these
// tests are about which suppression fires and what gets logged, which is exactly
// what has to keep working when someone later "simplifies" the early returns.
func hookOut(t *testing.T, home, root, prompt string) string {
	t.Helper()
	t.Setenv("HOME", home)
	in, err := json.Marshal(retrieveInput{Prompt: prompt, CWD: root})
	if err != nil {
		t.Fatal(err)
	}
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background()) // cobra: an unset context makes WithTimeout panic, which recover() would hide
	var out bytes.Buffer
	cmd.SetIn(bytes.NewReader(in))
	cmd.SetOut(&out)
	runRetrieveHook(cmd)
	return out.String()
}

func fireLog(t *testing.T, home, root string) []string {
	t.Helper()
	t.Setenv("HOME", home)
	idx, err := indexPathFor(root)
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(filepath.Dir(idx), "fires.log"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		t.Fatal(err)
	}
	var lines []string
	for _, l := range strings.Split(strings.TrimSpace(string(b)), "\n") {
		if l != "" {
			lines = append(lines, l)
		}
	}
	return lines
}

// Machine-injected text is 112 of 277 fires in the measured session — task
// notifications and system reminders, which carry no question. Ranking them is
// the bulk of the volume that trained the reader to skip the channel.
func TestHookSuppressesMachineInjectedText(t *testing.T) {
	home, root := t.TempDir(), t.TempDir()
	got := hookOut(t, home, root,
		"<task-notification>\n<task-id>abc</task-id>\nAgent finished\n</task-notification>")
	if got != "" {
		t.Errorf("machine text must not be ranked, got %q", got)
	}
	log := fireLog(t, home, root)
	if len(log) != 1 || !strings.Contains(log[0], "suppressed-machine") {
		t.Errorf("want one suppressed-machine record, got %v", log)
	}
}

// A one-word acknowledgement has nothing to retrieve against. The floor is in
// RUNES: a byte floor would suppress unevenly by language, which on this corpus
// means suppressing the majority language harder.
func TestHookSuppressesShortPromptsByRuneCount(t *testing.T) {
	for _, p := range []string{"devam", "evet", "ok", "tamam", "go on"} {
		home, root := t.TempDir(), t.TempDir()
		if got := hookOut(t, home, root, p); got != "" {
			t.Errorf("%q must not be ranked, got %q", p, got)
		}
		if log := fireLog(t, home, root); len(log) != 1 || !strings.Contains(log[0], "suppressed-short") {
			t.Errorf("%q: want suppressed-short, got %v", p, log)
		}
	}
	// …and anything at or above the floor must reach ranking. The second case is
	// the reason the floor counts runes: 12 accented characters are 12 runes but
	// 24 bytes, so a byte-based floor would admit it while suppressing a 12-rune
	// ASCII prompt — i.e. it would suppress unevenly by language, and this corpus
	// is read in two.
	for _, p := range []string{"why did we decide this", strings.Repeat("é", 12)} {
		home, root := t.TempDir(), t.TempDir()
		hookOut(t, home, root, p)
		if log := fireLog(t, home, root); len(log) != 0 {
			t.Errorf("%q must reach ranking, but was suppressed: %v", p, log)
		}
	}
}

// The tally is the subsystem's denominator, so its arithmetic is worth pinning:
// before this log existed, "does retrieval have anything to say, and does anyone
// read it?" could only be answered by a forensic pass over a 30 MB transcript.
func TestReadFireStatsTallies(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "fires.log")
	body := strings.Join([]string{
		"2026-08-03T00:00:00Z\tfired\ta.md b.md",
		"2026-08-03T00:01:00Z\tsilent",
		"2026-08-03T00:02:00Z\tsuppressed-machine",
		"2026-08-03T00:03:00Z\tsuppressed-short",
		"2026-08-03T00:04:00Z\tfired\ta.md",
		"", // trailing newline must not count as a fire
	}, "\n")
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	st, err := readFireStats(p)
	if err != nil {
		t.Fatal(err)
	}
	if st.Total != 5 || st.Fired != 2 || st.Silent != 1 || st.Machine != 1 || st.Short != 1 {
		t.Errorf("tally: %+v", st)
	}
	if st.Offered != 3 || st.Pages["a.md"] != 2 || st.Pages["b.md"] != 1 {
		t.Errorf("page counts: offered=%d pages=%v", st.Offered, st.Pages)
	}
}

// A missing log is "no fires yet", never an error — the log only exists once the
// hook has run since this shipped, and `atl retrieve stats` must not fail on a
// project that has simply never been queried.
func TestReadFireStatsMissingLogIsEmpty(t *testing.T) {
	st, err := readFireStats(filepath.Join(t.TempDir(), "nope.log"))
	if err != nil {
		t.Fatalf("missing log must not error: %v", err)
	}
	if st.Total != 0 {
		t.Errorf("want empty tally, got %+v", st)
	}
}

// Logging is best-effort by contract. The hook fails open on a missing model and
// a corrupt index; it must not start failing on an unwritable cache.
func TestLogRetrieveFireNeverPanicsOnUnwritableDir(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	// A regular file where the cache directory needs to be: MkdirAll must fail.
	if err := os.MkdirAll(filepath.Join(home, ".atl", "cache"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".atl", "cache", "retrieve"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	logRetrieveFire(t.TempDir(), "fired", []string{"a.md"}) // must simply return
}

// --- the ranked paths, exercised against a real index -----------------------

// buildTinyIndex writes a lexical-only index for `root` in a throwaway HOME, so
// the hook's ranked branches (fired / silent) can be driven without the embedder.
func buildTinyIndex(t *testing.T, home, root string, docs []retrieve.Doc) {
	t.Helper()
	t.Setenv("HOME", home)
	ix, err := retrieve.Build(context.Background(), docs, nil)
	if err != nil {
		t.Fatal(err)
	}
	p, err := indexPathFor(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := ix.Save(p); err != nil {
		t.Fatal(err)
	}
}

// The whole point of the floor: a prompt the corpus cannot answer produces
// nothing at all. Before this, the retriever had no way to express "no signal"
// and emitted its five nearest attractors instead.
func TestHookIsSilentWhenNothingMatches(t *testing.T) {
	home, root := t.TempDir(), t.TempDir()
	buildTinyIndex(t, home, root, []retrieve.Doc{
		{Path: "merge.md", Title: "Merge verification", Text: "the dispatch confirms a branch merged to base"},
	})
	if got := hookOut(t, home, root, "ornithology taxonomy of migratory seabirds"); got != "" {
		t.Errorf("no lexical overlap must yield nothing, got %q", got)
	}
	log := fireLog(t, home, root)
	if len(log) != 1 || !strings.Contains(log[0], "silent") {
		t.Errorf("want one silent record, got %v", log)
	}
}

// And the converse, so the suppressions cannot be "fixed" into suppressing
// everything: a real match still surfaces, and the log records what was offered.
func TestHookFiresAndLogsOfferedPaths(t *testing.T) {
	home, root := t.TempDir(), t.TempDir()
	buildTinyIndex(t, home, root, []retrieve.Doc{
		{Path: "merge.md", Title: "Merge verification", Text: "the dispatch confirms a branch merged to base"},
		{Path: "other.md", Title: "Unrelated", Text: "lentil soup with cumin"},
	})
	got := hookOut(t, home, root, "how does the dispatch confirm a branch merged")
	if !strings.Contains(got, "merge.md") {
		t.Errorf("a real match must surface, got %q", got)
	}
	log := fireLog(t, home, root)
	if len(log) != 1 || !strings.Contains(log[0], "\tfired\t") || !strings.Contains(log[0], "merge.md") {
		t.Errorf("want one fired record naming the offered path, got %v", log)
	}
}

// --- the tally a human reads ------------------------------------------------

func TestRenderFireStatsReportsTheDenominator(t *testing.T) {
	st := retrieveFireStats{
		Total: 10, Fired: 2, Silent: 3, Machine: 4, Short: 1, Offered: 3,
		Pages: map[string]int{"a.md": 2, "b.md": 1},
	}
	out := renderFireStats(st)
	// Assert the labelled line carries the right number, not its exact padding —
	// the arithmetic is the contract, the column widths are not.
	for label, want := range map[string]int{
		"fires": 10, "ranked": 5 /* fired+silent */, "offered": 2, "silent": 3, "suppressed": 5, /* machine+short */
	} {
		re := regexp.MustCompile(`(?m)^\s*` + label + `\s+(\d+)`)
		m := re.FindStringSubmatch(out)
		if m == nil {
			t.Errorf("no %q line in:\n%s", label, out)
			continue
		}
		if got, _ := strconv.Atoi(m[1]); got != want {
			t.Errorf("%s = %d, want %d", label, got, want)
		}
	}
	for _, want := range []string{"machine 4, short 1", "3 page refs", "distinct pages offered: 2", "a.md"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}

func TestRenderFireStatsSaysSoWhenEmpty(t *testing.T) {
	if out := renderFireStats(retrieveFireStats{}); !strings.Contains(out, "no fires recorded") {
		t.Errorf("an empty tally must say so, got %q", out)
	}
}

// More than ten distinct pages must not print a wall — the tally is read at a
// glance or it is not read.
func TestRenderFireStatsTruncatesThePageList(t *testing.T) {
	st := retrieveFireStats{Total: 20, Fired: 20, Pages: map[string]int{}}
	for i := 0; i < 14; i++ {
		st.Pages[fmt.Sprintf("p%02d.md", i)] = 20 - i
	}
	out := renderFireStats(st)
	if !strings.Contains(out, "… 4 more") {
		t.Errorf("want a truncation note, got:\n%s", out)
	}
}

// The command end to end, in a throwaway HOME and cwd — the glue between "where
// is the log" and "what does it say" is exactly what a refactor breaks silently,
// since both halves keep passing their own tests.
func TestRetrieveStatsCommandReadsThisProjectsLog(t *testing.T) {
	home, root := t.TempDir(), t.TempDir()
	t.Setenv("HOME", home)
	t.Chdir(root)

	logRetrieveFire(root, "fired", []string{"a.md", "b.md"})
	logRetrieveFire(root, "silent", nil)
	logRetrieveFire(root, "suppressed-machine", nil)

	cmd := retrieveStatsCmd
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatalf("stats: %v", err)
	}
	got := out.String()
	for _, want := range []string{"fires            3", "a.md", "distinct pages offered: 2"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
}

// A project that has never been queried must report that, not fail — `atl
// retrieve stats` is the first thing someone runs to find out whether the hook
// is doing anything at all.
func TestRetrieveStatsCommandOnAnUntouchedProject(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Chdir(t.TempDir())
	cmd := retrieveStatsCmd
	var out bytes.Buffer
	cmd.SetOut(&out)
	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatalf("stats on a fresh project must not error: %v", err)
	}
	if !strings.Contains(out.String(), "no fires recorded") {
		t.Errorf("got %q", out.String())
	}
}

// The log survives a line it does not understand — a future outcome name, or a
// truncated write — rather than mis-tallying it.
func TestReadFireStatsSkipsMalformedLines(t *testing.T) {
	p := filepath.Join(t.TempDir(), "fires.log")
	body := "not-a-record\n2026-08-03T00:00:00Z\tfired\ta.md\n2026-08-03T00:01:00Z\tsome-future-outcome\n"
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	st, err := readFireStats(p)
	if err != nil {
		t.Fatal(err)
	}
	if st.Fired != 1 || st.Total != 2 {
		t.Errorf("want 1 fired of 2 records, got %+v", st)
	}
}

// The lock working and the command CONSULTING it are two different things — a
// correct guard nothing calls is a shape this codebase has recorded three times.
// This drives the command with a held lock and asserts it declines.
func TestIndexCommandRefusesWhileAnotherBuildHoldsTheLock(t *testing.T) {
	home, root := t.TempDir(), t.TempDir()
	t.Setenv("HOME", home)
	t.Chdir(root)
	if err := os.MkdirAll(filepath.Join(root, ".atl", "wiki"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".atl", "wiki", "a.md"), []byte("# A\nbody\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	idx, err := indexPathFor(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(idx), 0o755); err != nil {
		t.Fatal(err)
	}
	// Our own pid: unambiguously alive, so the lock is believed.
	lock := filepath.Join(filepath.Dir(idx), "index.lock")
	if err := os.WriteFile(lock,
		[]byte(fmt.Sprintf("%d\n%d\n", os.Getpid(), time.Now().Unix())), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := retrieveIndexCmd
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetContext(context.Background())
	if err := cmd.Flags().Set("timeout", "30s"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatalf("a duplicate build must not be an error: %v", err)
	}
	if !strings.Contains(out.String(), "already running") {
		t.Errorf("the command must decline and say why, got %q", out.String())
	}
	// And it must not have written an index behind the lock.
	if _, err := os.Stat(idx); err == nil {
		t.Error("a refused build must not produce an index")
	}
}

// The other half of the same wiring: with no lock held the command must take
// it, build, and release — so a second build afterwards is not blocked by a
// lock the first one leaked.
func TestIndexCommandTakesAndReleasesTheLock(t *testing.T) {
	home, root := t.TempDir(), t.TempDir()
	t.Setenv("HOME", home)
	t.Chdir(root)
	if err := os.MkdirAll(filepath.Join(root, ".atl", "wiki"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".atl", "wiki", "a.md"),
		[]byte("# A\nsome body text\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := retrieveIndexCmd
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetContext(context.Background())
	if err := cmd.Flags().Set("timeout", "120s"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatalf("build failed: %v", err)
	}
	if strings.Contains(out.String(), "already running") {
		t.Fatalf("nothing held the lock, got %q", out.String())
	}

	idx, err := indexPathFor(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(idx), "index.lock")); !os.IsNotExist(err) {
		t.Error("the lock must be released when the build returns")
	}
}

// A translated prompt writes TWO lines — "translated" before the search, then the
// search's own outcome — so counting the first as a fire inflates the denominator
// every percentage in this report is computed against, and inflates it further the
// more translation is used. The report exists to decide whether injected context
// works as a delivery channel at all; a denominator that drifts with adoption
// would mis-decide exactly that.
func TestFireStatsDoesNotCountATranslationAsAPrompt(t *testing.T) {
	p := filepath.Join(t.TempDir(), "fires.log")
	if err := os.WriteFile(p, []byte(
		"t\ttranslated\n"+
			"t\tfired\ta.md\n"+
			"t\tsuppressed-machine\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	st, err := readFireStats(p)
	if err != nil {
		t.Fatal(err)
	}
	if st.Total != 2 {
		t.Fatalf("Total = %d, want 2 — the translation line is not a prompt", st.Total)
	}
	if st.Translated != 1 {
		t.Fatalf("Translated = %d, want 1 — it must still be reported, just not as a fire", st.Translated)
	}
	// The rendered percentages must agree with the corrected denominator: one
	// ranked fire out of two prompts is 50%, not the 33% an inflated Total gives.
	if out := renderFireStats(st); !strings.Contains(out, " 50.0%") {
		t.Fatalf("percentages still use the inflated denominator:\n%s", out)
	}
}

// The translator is a real `claude -p` session, so it fires the SAME
// UserPromptSubmit hook this code runs inside. Unmarked, a non-English prompt
// searched twice — once against the translation instruction — and the extra fire
// carried its own page refs, inflating the offer count that the follow-through
// metric divides by. Proven in the wild by line count: one prompt wrote
// fired/translated/fired with a credential present and a bare fired without one.
//
// The probe is a machine prompt because that path logs unconditionally, without
// needing an index. That isolates the new guard from the no-index early return —
// the first version of this test used an ordinary prompt and its control failed,
// because in a throwaway project nothing is logged for one anyway.
func TestInternalSessionWritesNoFireAtAll(t *testing.T) {
	const machine = "<task-notification>a background task finished</task-notification>"

	home, root := t.TempDir(), t.TempDir()
	t.Setenv(atlInternalSessionEnv, "1")
	if out := hookOut(t, home, root, machine); out != "" {
		t.Fatalf("injected context into an internal session: %q", out)
	}
	if lines := fireLog(t, home, root); len(lines) != 0 {
		t.Fatalf("recorded %d fire(s) for a session with no human reader: %v", len(lines), lines)
	}

	// The control. Without it this passes just as well against a hook that stopped
	// logging entirely — the guard has to be narrow, not merely present.
	home2, root2 := t.TempDir(), t.TempDir()
	t.Setenv(atlInternalSessionEnv, "")
	hookOut(t, home2, root2, machine)
	if lines := fireLog(t, home2, root2); len(lines) != 1 {
		t.Fatalf("ordinary session logged %v, want exactly one suppressed-machine fire", lines)
	}
}
