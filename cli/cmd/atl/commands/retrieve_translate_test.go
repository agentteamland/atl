package commands

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
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
	// HOME is isolated because there is now a FILE source too. Without this the
	// test reads the developer's real credential file and passes on CI while
	// failing on the machine of anyone who actually configured the feature — the
	// inverse of the CI-is-poorer asymmetry this function's own comment records,
	// and the direction a green suite cannot see.
	t.Setenv("HOME", t.TempDir())
	t.Setenv("CLAUDE_CODE_OAUTH_TOKEN", "")
	t.Setenv("ANTHROPIC_API_KEY", "")
	if claudeAvailable() {
		t.Fatal("reported available with no credential in any source")
	}
}

// The file is a real source, not a decoration. It exists because the environment
// cannot always be reached: the host withholds its own credential from the
// processes it starts, so the documented place to set a variable cannot deliver
// this one to a hook.
func TestCredentialResolvesFromTheFileWhenTheEnvironmentIsEmpty(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("CLAUDE_CODE_OAUTH_TOKEN", "")
	t.Setenv("ANTHROPIC_API_KEY", "")
	writeCredFile(t, home, "  sk-ant-oat01-from-file\n")

	cred, ok := translatorCredential()
	if !ok {
		t.Fatal("a populated credential file was not recognised as a credential")
	}
	// Trimming matters: the file is hand-written, so a trailing newline is the
	// normal case rather than the exception, and an untrimmed token is rejected by
	// the API in a way that surfaces as an expired credential.
	if cred.Value != "sk-ant-oat01-from-file" {
		t.Errorf("Value = %q — surrounding whitespace must be trimmed", cred.Value)
	}
	if cred.EnvName != "CLAUDE_CODE_OAUTH_TOKEN" {
		t.Errorf("EnvName = %q, want the variable the child actually reads", cred.EnvName)
	}
	if cred.Source != translatorCredSourceFile || cred.FromEnv {
		t.Errorf("Source = %q FromEnv = %v — the expired notice sends the user to Source, so it must name the file", cred.Source, cred.FromEnv)
	}
}

// Environment wins: it is the per-invocation, more specific channel, and
// silently preferring a file over a value the user explicitly exported is the
// more surprising of the two orders.
func TestEnvironmentWinsOverTheFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("CLAUDE_CODE_OAUTH_TOKEN", "sk-ant-oat01-from-env")
	writeCredFile(t, home, "sk-ant-oat01-from-file")

	cred, ok := translatorCredential()
	if !ok {
		t.Fatal("no credential resolved with both sources present")
	}
	if cred.Value != "sk-ant-oat01-from-env" {
		t.Errorf("Value = %q, want the exported one", cred.Value)
	}
	// Source is what the expired notice points the user at. Naming the file when
	// the environment is in use sends them to edit something that is not being
	// read — the exact drift this change exists to end.
	if cred.Source != "CLAUDE_CODE_OAUTH_TOKEN" {
		t.Errorf("Source = %q, want the env var actually in use", cred.Source)
	}
}

// A created-but-unfilled file is not a credential. An existence check would
// report configured, suppress the actionable missing-credential notice, and
// then — three prompts later — announce that a credential the user never had
// has expired.
func TestAnEmptyCredentialFileIsNotACredential(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("CLAUDE_CODE_OAUTH_TOKEN", "")
	t.Setenv("ANTHROPIC_API_KEY", "")
	writeCredFile(t, home, "\n   \n")

	if _, ok := translatorCredential(); ok {
		t.Fatal("an empty credential file was reported as a credential")
	}
}

// The remedy has to name the place the user actually has to edit.
func TestExpiredNoticeNamesTheSourceInUse(t *testing.T) {
	if msg := expiredTranslationNotice(translatorCredSourceFile); !strings.Contains(msg, translatorCredSourceFile) {
		t.Errorf("file-configured user is not sent to the file:\n%s", msg)
	}
	msg := expiredTranslationNotice("CLAUDE_CODE_OAUTH_TOKEN")
	if !strings.Contains(msg, ".zshenv") {
		t.Errorf("env-configured user is not sent to a shell file that a hook actually reads:\n%s", msg)
	}
	if strings.Contains(msg, translatorCredSourceFile) {
		t.Errorf("env-configured user sent to edit a file that is not being read:\n%s", msg)
	}
}

// writeCredFile lays down the translator credential file under an isolated HOME.
func writeCredFile(t *testing.T, home, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(home, ".atl"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".atl", translatorCredFile), []byte(body), 0o600); err != nil {
		t.Fatal(err)
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
	for _, want := range []string{"setup-token", translatorCredSourceFile, ".zshenv"} {
		if !strings.Contains(msg, want) {
			t.Errorf("notice does not tell the user how to act: missing %q", want)
		}
	}
	// The two places that do NOT work must be named as warnings, BEFORE the two
	// that do. Both halves are load-bearing and this is what pins them.
	//
	// Naming them at all is the point of the notice: measured on macOS, the host
	// strips this variable from every process it spawns, so settings.json's env
	// block cannot deliver it to a hook, and zsh reads .zshrc only for interactive
	// shells, which a hook is not. A reader who already tried one of those and saw
	// the notice again concludes the notice is wrong and stops reading.
	//
	// And the ORDER is what stops them being read as suggestions: a reader
	// skimming for a path must not meet ~/.zshrc first.
	offer := strings.Index(msg, "EITHER of these")
	if offer < 0 {
		t.Fatalf("notice no longer offers the working locations as a choice:\n%s", msg)
	}
	for _, doesNotWork := range []string{"~/.zshrc", "settings.json"} {
		i := strings.Index(msg, doesNotWork)
		if i < 0 {
			t.Errorf("notice does not warn about %q, so a reader who used it learns nothing", doesNotWork)
			continue
		}
		if i > offer {
			t.Errorf("%q appears after the offer — it reads as a place to put the token, which is the failure this notice exists to prevent", doesNotWork)
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
	if _, out, _ := translatePrompt(t.Context(), strings.Repeat("z", maxTranslatableRunes+1)); out != translateSkipped {
		t.Fatalf("outcome = %v, want translateSkipped — the cap is a deliberate skip, and recording it as a failure would count toward 'your credential expired'", out)
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
	cred := translatorCred{EnvName: "CLAUDE_CODE_OAUTH_TOKEN", Value: "sk-ant-oat01-test", Source: translatorCredSourceFile}
	cmd := translateCommand(t.Context(), "ogrenme kuyrugu isaretcileri", cred)

	var found, carried bool
	for _, kv := range cmd.Env {
		if kv == atlInternalSessionEnv+"=1" {
			found = true
		}
		if kv == cred.EnvName+"="+cred.Value {
			carried = true
		}
	}
	if !found {
		t.Fatalf("child not marked internal — its hook will fire; env had %d entries", len(cmd.Env))
	}
	// The credential has to REACH the child, not merely be resolvable by the
	// parent. A file-borne token is absent from os.Environ() by definition, so
	// without this the resolver would report the translator available while every
	// attempt paid a process spawn and up to the timeout and then failed — on
	// exactly the prompts the feature exists for.
	if !carried {
		t.Fatal("credential not passed to the child: it reads its token from the environment, and a file-borne one reaches it only through this Env")
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

// The expired branch end to end, not just its predicate. It has to resolve a
// credential in order to name the source in the remedy, and a resolution that
// comes back empty returns silence — so this path can fail by saying NOTHING,
// which is indistinguishable from a healthy machine. The unit tests above cover
// translationFailing and the notice text separately; only this one covers the
// wiring between them.
func TestRetrievalNoticeReportsAnExpiredFileCredential(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("CLAUDE_CODE_OAUTH_TOKEN", "")
	t.Setenv("ANTHROPIC_API_KEY", "")
	writeCredFile(t, home, "sk-ant-oat01-stale")

	// The real probe needs a `claude` binary, which CI does not have.
	orig := claudeAvailable
	claudeAvailable = func() bool { return true }
	t.Cleanup(func() { claudeAvailable = orig })

	project := t.TempDir()
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
	writeFires(t, idx, "translate-failed", "translate-failed", "translate-failed")

	msg, ok := retrievalTranslationNotice(project)
	if !ok {
		t.Fatal("a dead credential produced no notice at all — the failure mode here is silence, which looks exactly like a healthy machine")
	}
	if !strings.Contains(msg, translatorCredSourceFile) {
		t.Errorf("the remedy does not name the file the credential actually came from:\n%s", msg)
	}
}

// A skipped translation must be transparent to the expiry evidence. Three
// ordinary English prompts in a row are completely normal, and before the
// outcome was split they were logged identically to three 401s — telling a user
// with a perfectly good token that it had expired.
func TestSkippedTranslationsAreNotEvidenceOfAnExpiry(t *testing.T) {
	idx := filepath.Join(t.TempDir(), "index.gob")
	writeFires(t, idx, "translate-skipped", "translate-skipped", "translate-skipped", "translate-skipped")
	if translationFailing(idx) {
		t.Error("skipped translations read as a dead credential — an English-speaking user would be told their token expired")
	}

	// …and they do not interrupt a genuine run of failures either: the skips are
	// invisible to the count rather than resetting it.
	idx2 := filepath.Join(t.TempDir(), "index.gob")
	writeFires(t, idx2, "translate-failed", "translate-skipped", "translate-failed", "translate-skipped", "translate-failed")
	if !translationFailing(idx2) {
		t.Error("three real failures stopped counting because skips sat between them")
	}
}

// The classifier's whole reason for existing is that four conditions used to be one
// number with one guess attached. So the test is a table of all four plus the honest
// fifth, and the arm that matters most is the LAST one: an unreadable failure must
// say it is unreadable rather than pick the likeliest, because picking the likeliest
// is precisely the defect being removed.
func TestClassifyTranslateFailureNamesTheConditionOrAdmitsItCannot(t *testing.T) {
	deadline, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()
	live := context.Background()

	for _, c := range []struct {
		name   string
		ctx    context.Context
		err    error
		output string
		want   translateFailure
	}{
		// Structural. Read from Go's own values, so they hold whatever language the
		// child speaks — which is why they are checked first.
		{"a fired deadline", deadline, errors.New("signal: killed"), "", failTimeout},
		{"the binary is missing", live, exec.ErrNotFound, "", failNoBinary},

		// A killed subprocess still prints whatever it had reached. Without the
		// deadline being checked FIRST, that fragment would classify the timeout as
		// whatever it happened to look like.
		{"a deadline that also printed", deadline, errors.New("killed"), "You've hit your weekly limit", failTimeout},

		// Text-matched. The first is the exact string the live machine produced.
		{"the real quota message", live, errors.New("exit status 1"),
			"You've hit your weekly limit · resets 8pm (Europe/Istanbul)", failQuota},
		{"another quota phrasing", live, errors.New("exit status 1"), "Usage limit reached", failQuota},
		{"a refused credential", live, errors.New("exit status 1"), "Invalid API key · Please run /login", failAuth},
		{"an expired one", live, errors.New("exit status 1"), "OAuth access token has expired", failAuth},

		// The load-bearing arm. Nothing recognisable came back, so the answer is
		// "I cannot say" — not the most likely of the four.
		{"something nobody anticipated", live, errors.New("exit status 3"), "panic: interface conversion", failUnclassified},
		{"no output at all", live, errors.New("exit status 1"), "", failUnclassified},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got := classifyTranslateFailure(c.ctx, c.err, c.output); got != c.want {
				t.Errorf("classify(%q) = %q, want %q", c.output, got, c.want)
			}
		})
	}
}

// Matching is case-insensitive because the child's wording is not ours to pin. This
// is the weakest arm by construction — a message match is a claim about
// human-readable output, and human-readable output is localised. The two structural
// arms exist so the conditions that HAVE a machine-readable signal never depend on
// this one.
func TestClassifyTranslateFailureIgnoresCase(t *testing.T) {
	if got := classifyTranslateFailure(context.Background(), errors.New("x"), "WEEKLY LIMIT REACHED"); got != failQuota {
		t.Errorf("got %q, want %q — the match must not depend on capitalisation", got, failQuota)
	}
}

// A translation survives the process that produced it, which is the whole point: the
// mechanism spends the user's own usage budget, so translating the same sentence twice
// is spending it twice for one answer.
func TestTranslationCacheRoundTrips(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	const prompt = "profil deposu nerede duruyor"

	if _, hit := cachedTranslation(prompt); hit {
		t.Fatal("an empty cache reported a hit")
	}
	storeTranslation(prompt, "where the profile store lives")
	got, hit := cachedTranslation(prompt)
	if !hit || got != "where the profile store lives" {
		t.Fatalf("round trip failed: got %q hit=%v", got, hit)
	}
	// A different prompt is a different entry. Obvious, and it is the assertion that
	// would catch a key derived from something constant.
	if _, hit := cachedTranslation("bambaska bir soru"); hit {
		t.Error("an unrelated prompt hit the cache")
	}
}

// The key is derived from the INSTRUCTION, not from the prompt alone.
//
// That is what makes a change to the instruction invalidate every stored answer. A key
// over the prompt alone would keep serving answers to a question no longer being asked,
// and nothing would report it — the entries would still be well-formed. This project has
// measured the same failure in an index whose vectors outlived the model that made them.
func TestTranslationKeyCoversTheInstructionSoAChangeInvalidatesIt(t *testing.T) {
	const prompt = "oturum baslangici"
	sum := sha256.Sum256([]byte(prompt))
	if translationKey(prompt) == hex.EncodeToString(sum[:]) {
		t.Error("the key is a hash of the prompt alone — rewording the instruction would " +
			"leave every cached answer in place, answering the old question")
	}
}

// An empty answer is not an answer. Storing one would turn a single bad response into a
// permanent one, which is the specific way a cache makes things worse rather than
// merely failing to help.
func TestTranslationCacheRefusesAnEmptyAnswer(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	storeTranslation("bir soru", "   \n ")
	if _, hit := cachedTranslation("bir soru"); hit {
		t.Error("an empty translation was cached")
	}
}

// The maintainer runs several projects at once, so concurrent writers are the ordinary
// case rather than an edge. One file per entry plus an atomic rename is what makes that
// safe; a single shared file would be read-modify-written and the loser's entry would
// vanish with no error.
func TestTranslationCacheSurvivesConcurrentWriters(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	done := make(chan struct{})
	for i := 0; i < 8; i++ {
		go func(n int) {
			defer func() { done <- struct{}{} }()
			p := "soru " + string(rune('a'+n))
			storeTranslation(p, "query "+string(rune('a'+n)))
		}(i)
	}
	for i := 0; i < 8; i++ {
		<-done
	}
	for i := 0; i < 8; i++ {
		p := "soru " + string(rune('a'+i))
		got, hit := cachedTranslation(p)
		if !hit || got != "query "+string(rune('a'+i)) {
			t.Errorf("%q lost or corrupted: got %q hit=%v", p, got, hit)
		}
	}
	// No temporary file may survive. A leftover .tmp-* is a half-written answer sitting
	// where a reader could one day be pointed at it.
	dir, _ := translationCacheDir()
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".tmp-") {
			t.Errorf("a temporary file survived: %s", e.Name())
		}
	}
}

// A cache hit is served by translatePrompt itself, with NO credential present.
//
// This is the assertion the tests above cannot make. They call the cache functions
// directly, so they pin the resolver and are structurally incapable of noticing that
// the call path never consults it — which a revert arm demonstrated: neutralising the
// lookup inside translatePrompt reddened nothing.
//
// The absent credential is what makes this test say something worth saying. With no
// credential no subprocess can run, so a returned translation can only have come from
// the cache — and that is precisely the state a user is in when their weekly limit is
// exhausted. The cache is worth most exactly when the translator is unavailable, and
// this is the arm that proves it works there.
func TestTranslatePromptServesTheCacheWithoutACredential(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("CLAUDE_CODE_OAUTH_TOKEN", "")
	t.Setenv("ANTHROPIC_API_KEY", "")
	const prompt = "profil deposu nerede duruyor"
	storeTranslation(prompt, "where the profile store lives")

	got, outcome, why := translatePrompt(context.Background(), prompt)
	if outcome != translateCached {
		t.Fatalf("outcome = %v, want translateCached (why=%q) — with no credential the "+
			"only possible source is the cache", outcome, why)
	}
	if got != "where the profile store lives" {
		t.Errorf("got %q, want the cached query", got)
	}
}

// And the miss path still degrades the way it did before: no credential, no cache
// entry, nothing runs. The cache must not have turned an honest skip into anything else.
func TestTranslatePromptStillSkipsOnAMissWithNoCredential(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("CLAUDE_CODE_OAUTH_TOKEN", "")
	t.Setenv("ANTHROPIC_API_KEY", "")

	if _, outcome, _ := translatePrompt(context.Background(), "hic gorulmemis bir soru"); outcome != translateSkipped {
		t.Errorf("outcome = %v, want translateSkipped", outcome)
	}
}
