package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Fixtures are written without diacritics on purpose. They only need to be
// NON-ENGLISH — what is under test is the validation of what comes back, not the
// input's orthography — and committed artifacts here are English-only, so the
// accented forms a user would really type cannot live in the repo.
//
// The translator is an LLM, so it can answer the question instead of rewriting
// it, refuse, apologise, or explain itself. Every one of those is a WORSE query
// than the original, and the hook's contract is that translation may improve
// retrieval and must never make it worse — so anything that does not look like a
// short query has to be rejected rather than used.
func TestCleanTranslationRejectsWhatIsNotAQuery(t *testing.T) {
	const original = "ogrenme kuyrugu isaretcileri nasil tekillestiriyor"
	cases := []struct {
		name, raw string
	}{
		{"empty", "   "},
		{"an explanation instead of a translation", "I'd be happy to help! The text appears to be Turkish. It is asking about how the learning queue deduplicates markers across sessions, which relates to content hashing and tombstones in the queue implementation."},
		{"a refusal", "I cannot translate this text because it contains no clear technical terms that I can map to English search vocabulary without more context about your system."},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got, ok := cleanTranslation(c.raw, original); ok {
				t.Fatalf("accepted a non-query answer: %q", got)
			}
		})
	}
}

func TestCleanTranslationAcceptsAQuery(t *testing.T) {
	got, ok := cleanTranslation("  \"learning queue dedup markers across sessions\"  \n",
		"ogrenme kuyrugu isaretcileri nasil tekillestiriyor")
	if !ok {
		t.Fatal("rejected a well-formed query")
	}
	if got != "learning queue dedup markers across sessions" {
		t.Fatalf("did not strip quotes/whitespace: %q", got)
	}
}

// An English prompt comes back unchanged, and re-running the search with an
// identical query is pure waste — so an unchanged answer is reported as "no
// translation" rather than as a successful one.
func TestCleanTranslationTreatsAnUnchangedAnswerAsNoTranslation(t *testing.T) {
	const en = "how does the learning queue dedup markers"
	if _, ok := cleanTranslation(en, en); ok {
		t.Fatal("an unchanged answer must not count as a translation")
	}
	if _, ok := cleanTranslation("  How Does The Learning Queue Dedup Markers \n", en); ok {
		t.Fatal("case/whitespace differences are not a translation either")
	}
}

// Only the first line survives. A model that emits the query and then adds a
// note would otherwise inject its prose into the search terms.
func TestCleanTranslationKeepsOnlyTheFirstLine(t *testing.T) {
	got, ok := cleanTranslation("marker hash dedup queue\n\nNote: I kept the identifiers unchanged as requested.", "isaretci tekillestirme")
	if !ok || got != "marker hash dedup queue" {
		t.Fatalf("got %q ok=%v, want the first line alone", got, ok)
	}
}

// No credential means no translator. Checked without spending an API call —
// the hook fires on every prompt and must not pay for a liveness probe.
func TestClaudeAvailableRequiresACredential(t *testing.T) {
	t.Setenv("CLAUDE_CODE_OAUTH_TOKEN", "")
	t.Setenv("ANTHROPIC_API_KEY", "")
	if claudeAvailable() {
		t.Fatal("reported available with no credential in the environment")
	}
}

// The notice is information, not a gate — but it must not appear where it would
// mean nothing. Silence in the irrelevant cases is what keeps it worth reading
// when it does appear.
func TestTranslationNoticeOnlyWhereItWouldChangeSomething(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	project := t.TempDir()

	// Stated, not inherited: the real probe needs a `claude` binary on PATH, which
	// CI does not have — so an environment-driven version of this test asserts a
	// state the runner cannot reach, and passes only on a developer's machine.
	available := false
	orig := claudeAvailable
	claudeAvailable = func() bool { return available }
	t.Cleanup(func() { claudeAvailable = orig })

	if _, ok := retrievalTranslationNotice(project); ok {
		t.Fatal("shown in a project with no index — nothing to translate against")
	}

	idx, err := indexPathFor(project)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(idx), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(idx, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	msg, ok := retrievalTranslationNotice(project)
	if !ok {
		t.Fatal("not shown where retrieval is live and no credential is configured")
	}
	for _, want := range []string{"setup-token", "CLAUDE_CODE_OAUTH_TOKEN"} {
		if !strings.Contains(msg, want) {
			t.Errorf("notice does not tell the user how to act: missing %q", want)
		}
	}
	// It must read as optional. A notice that sounds like a failure is a gate in
	// everything but name, and the decision was explicit that this is not one.
	if !strings.Contains(msg, "not a requirement") {
		t.Error("notice does not say it is optional")
	}

	available = true
	if _, ok := retrievalTranslationNotice(project); ok {
		t.Fatal("still shown after the credential is configured — that is noise")
	}
}

// An oversized prompt is a paste, not a question: the lexical arm already has
// plenty to match on, and sending it costs latency for nothing.
func TestTranslateSkipsAnOversizedPrompt(t *testing.T) {
	if _, ok := translatePrompt(t.Context(), strings.Repeat("z", maxTranslatableRunes+1)); ok {
		t.Fatal("attempted to translate a prompt past the cap")
	}
}

// The subprocess must carry the internal-session mark, or the child session's own
// UserPromptSubmit hook searches the corpus again and records a fire nobody read.
//
// Asserted on the constructed command rather than on behaviour, because the
// behaviour needs a real `claude` binary. That matters: a mutation removing the
// marking left every other test in this package green — the guard that CONSUMES
// the mark was covered, the wiring that SETS it was not.
func TestTranslateCommandMarksTheChildAsInternal(t *testing.T) {
	cmd := translateCommand(t.Context(), "ogrenme kuyrugu isaretcileri")

	var found bool
	for _, kv := range cmd.Env {
		if kv == atlInternalSessionEnv+"=1" {
			found = true
		}
	}
	if !found {
		t.Fatalf("child not marked internal — its hook will fire; env had %d entries", len(cmd.Env))
	}
	// The mark is an addition, not a replacement: the child still needs the
	// credential and PATH it inherits, so a bare one-entry Env would break it.
	if len(cmd.Env) < 2 {
		t.Fatalf("child env replaced rather than extended: %v", cmd.Env)
	}
}

// writeFires lays down a fire log with the given outcomes, oldest first.
func writeFires(t *testing.T, idxPath string, outcomes ...string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(idxPath), 0o755); err != nil {
		t.Fatal(err)
	}
	var b strings.Builder
	for _, o := range outcomes {
		b.WriteString("2026-08-06T00:00:00Z\t" + o + "\n")
	}
	if err := os.WriteFile(filepath.Join(filepath.Dir(idxPath), "fires.log"), []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}
}

// The state this exists to detect: a credential that is SET and no longer valid.
// Without it the env var is present, the notice stays suppressed, and every
// non-English prompt silently searches on one arm.
func TestThreeConsecutiveFailuresReadAsAnExpiredCredential(t *testing.T) {
	idx := filepath.Join(t.TempDir(), "index.gob")
	writeFires(t, idx, "fired", "translate-failed", "translate-failed", "translate-failed")
	if !translationFailing(idx) {
		t.Error("three consecutive translation failures must read as a dead credential")
	}
}

// One or two failures must stay silent. A timeout is not an expiry, and a notice
// that fires on a blip is the constant channel this project has measured.
func TestAFewFailuresAreNotAnExpiry(t *testing.T) {
	for _, n := range []int{1, 2} {
		idx := filepath.Join(t.TempDir(), "index.gob")
		out := []string{"translated"}
		for i := 0; i < n; i++ {
			out = append(out, "translate-failed")
		}
		if translationFailing(idx) {
			t.Errorf("%d failure(s) must not read as an expiry", n)
		}
		writeFires(t, idx, out...)
		if translationFailing(idx) {
			t.Errorf("%d failure(s) after a success must not read as an expiry", n)
		}
	}
}

// A success anywhere in the recent window clears it — the credential works.
func TestARecentSuccessClearsTheSignal(t *testing.T) {
	idx := filepath.Join(t.TempDir(), "index.gob")
	writeFires(t, idx, "translate-failed", "translate-failed", "translate-failed", "translated")
	if translationFailing(idx) {
		t.Error("a success after the failures means the credential works")
	}
}

// Unrelated outcomes between failures must not break the run: the log is shared
// with every retrieval fire, and translation attempts are a minority of it.
func TestUnrelatedFiresDoNotInterruptTheCount(t *testing.T) {
	idx := filepath.Join(t.TempDir(), "index.gob")
	writeFires(t, idx, "translate-failed", "fired", "translate-failed", "fired", "translate-failed")
	if !translationFailing(idx) {
		t.Error("interleaved unrelated fires must not hide three consecutive failures")
	}
}

// No log, no signal. This decides whether to PRINT something, so being wrong
// must cost nothing.
func TestNoFireLogIsSilent(t *testing.T) {
	if translationFailing(filepath.Join(t.TempDir(), "index.gob")) {
		t.Error("a missing fire log must not produce a notice")
	}
}
