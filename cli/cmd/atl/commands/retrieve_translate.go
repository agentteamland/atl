package commands

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/agentteamland/atl/cli/internal/doctor"
	"github.com/agentteamland/atl/cli/internal/scope"
)

// The corpus is English — ATL's own rule is that every committed artifact is
// English — while a user may well work in another language. That asymmetry has
// one measurable consequence: BM25 shares no token with a non-English query, so
// the lexical arm scores a mathematical zero and retrieval runs on the semantic
// arm alone.
//
// Measured against a fixed answer key (12 questions asked in two languages,
// recall@5 of the page that IS the answer):
//
//	English   75%     both arms
//	Turkish   25%     semantic arm only
//
// and the semantic arm alone scores IDENTICALLY in both languages (25%), so the
// entire gap is the missing lexical arm — not comprehension.
//
// Translating the query is the field's standard answer: cross-lingual lexical
// retrieval works only when query and corpus share a language, and the only two
// ways to arrange that are translating the query or translating the corpus.
//
// The expected gain is measured rather than projected: because the answer key
// holds every question in both languages, a perfectly translated query scores
// what its English twin scores — 75% against today's 25%.

const (
	// translateTimeout bounds the translation subprocess.
	//
	// DERIVED, not chosen. The binding constraint is not the latency of a
	// translation — it is the harness's own ceiling on the hook this runs inside,
	// which nothing in this repository had measured or recorded.
	//
	// Claude Code cancels a `command` hook at a default that UserPromptSubmit
	// lowers to 30 SECONDS, and `atl retrieve` is registered as exactly that with
	// no `timeout` of its own (settings.Hook cannot express one). So 30s is the
	// whole budget for translation AND everything after it.
	//
	// The rest of the hook, measured warm on this machine, costs ~2.5s; its worst
	// case is bounded by the 5s context around the embedder and the query, plus a
	// cold index load — call it ~7s. That leaves ~23s for translation, and 20s
	// keeps a margin under it.
	//
	// Why not raise it toward the ceiling, which the observed distribution invites:
	// translation measured 9–24s over nine samples (median 16), and a 92s success
	// was measured by hand. Those longer calls cannot be served from here at ANY
	// value of this constant — and raising it trades a VISIBLE failure for an
	// INVISIBLE one. A translation that hits this deadline logs
	// `translate-failed:timeout` and the prompt still searches untranslated; a hook
	// the harness kills at 30s logs NOTHING and loses retrieval entirely, which is
	// both the worse outcome and the unobservable one.
	//
	// So the tail is not served by a bigger number. It is served by the cache
	// above: a query translated once is free every time after, including while the
	// weekly limit is exhausted.
	translateTimeout = 20 * time.Second

	// maxTranslatableRunes caps what is sent. A retrieval query is a question,
	// not a document; something far longer is a paste, where the lexical arm has
	// plenty to match on already and translation buys nothing.
	maxTranslatableRunes = 600
)

// translatePrompt asks the user's own Claude for an English search query.
//
// Anything other than translateOK leaves the caller searching with the original
// prompt. The hook is fail-open by contract: translation may improve retrieval
// and must never be able to make it worse than not having tried.
//
// The outcome is three-valued rather than a bool because one of the three is
// evidence about the CREDENTIAL and the other two are not, and the expired-
// credential notice is built on that evidence. Collapsing them told a user with
// a perfectly good token that it had expired, because three English prompts in a
// row are rejected by cleanTranslation — a completely ordinary thing to happen —
// and were logged identically to three 401s.
//
// `claude -p` is invoked with NO --model flag on purpose. The user's own default
// model is used, so no model name exists in this codebase to go stale when a
// model is retired — a requirement of the decision, not an omission.
func translatePrompt(ctx context.Context, prompt string) (string, translateOutcome, translateFailure) {
	if len([]rune(prompt)) > maxTranslatableRunes {
		return "", translateSkipped, ""
	}
	// The cache is consulted BEFORE the credential, and that ordering is deliberate.
	// A hit needs no credential, spawns no subprocess and spends no quota, so a
	// machine whose limit is exhausted still gets its translation for anything it has
	// translated before — which is exactly when it is most needed and least available.
	if q, hit := cachedTranslation(prompt); hit {
		return q, translateCached, ""
	}
	cred, ok := translatorCredential()
	if !ok {
		// The credential was there when claudeAvailable ran and is gone now. Skipped,
		// not failed: no subprocess ran, so nothing was observed about a credential's
		// validity — and the next prompt will find claudeAvailable false and print the
		// missing-credential notice, which is the correct one for this state.
		return "", translateSkipped, ""
	}
	ctx, cancel := context.WithTimeout(ctx, translateTimeout)
	defer cancel()

	cmd := translateCommand(ctx, prompt, cred)
	var out, errOut bytes.Buffer
	cmd.Stdout = &out
	// stderr is captured and NEVER PRINTED, which is the original reason and it
	// still holds: this runs inside a hook whose stdout is injected into the
	// agent's context, and a stray warning from the child would land there as if it
	// were retrieved knowledge.
	//
	// What changed is that it is no longer DISCARDED. The buffer was called
	// `discard` and thrown away, so every failure below collapsed into one counter
	// and the stats line had to guess at the cause — it printed "credential
	// expired?" over a quota exhaustion and sent the diagnosis to the wrong place.
	// The text is read here to classify, and goes nowhere else.
	cmd.Stderr = &errOut
	if err := cmd.Run(); err != nil {
		// Several different conditions land here — a rejected credential, an
		// exhausted quota, a timeout, a missing binary — and they have different
		// remedies. Classified rather than counted; see classifyTranslateFailure.
		return "", translateFailed, classifyTranslateFailure(ctx, err, out.String()+errOut.String())
	}
	q, ok := cleanTranslation(out.String(), prompt)
	if ok {
		// Only a validated query is cached. cleanTranslation is what rejects an
		// explanation, a refusal or an unchanged echo, so storing before it would
		// make a bad answer permanent — a cache turns a transient wrong into a
		// standing one.
		storeTranslation(prompt, q)
	}
	if !ok {
		// The subprocess ran fine and its answer was deliberately not used: already
		// English, empty, or prose instead of a query. That says nothing whatever
		// about the credential, so it must not be logged as though it did.
		return "", translateSkipped, ""
	}
	return q, translateOK, ""
}

// translateOutcome distinguishes the three things that can happen, because only
// one of them is evidence about the credential.
// translationCacheDir is where translated queries are kept.
//
// GLOBAL, not per project. A translation is a pure function of its input — the same
// Turkish phrase becomes the same English query in every repository — so a per-project
// cache would fragment it and re-spend the usage budget once per project for the same
// sentence. The store it caches against is global for the same reason.
func translationCacheDir() (string, error) {
	layer, err := scope.LayerDir(scope.Global, "")
	if err != nil {
		return "", err
	}
	return filepath.Join(layer, "cache", "translate"), nil
}

// translationKey identifies a cached translation.
//
// It hashes the INSTRUCTION as well as the prompt, and that is the load-bearing half.
// A cached value is only valid for the question that produced it, so changing the
// instruction — a new rule, a reworded constraint — must invalidate every entry rather
// than silently keep serving answers to a question no longer being asked. Writing the
// producer's identity into the artifact is the alternative to remembering to bump a
// version by hand, and this project has measured what happens when that is left to
// memory.
//
// Hashed rather than stored: the prompt is the user's own text and does not belong in a
// filename, in a log, or anywhere a directory listing can be read.
func translationKey(prompt string) string {
	sum := sha256.Sum256([]byte(translationInstruction(prompt)))
	return hex.EncodeToString(sum[:])
}

// cachedTranslation returns a previously translated query, if there is one.
//
// Every failure is a MISS rather than an error. The cache is an optimisation over a
// mechanism that is already fail-open; a cache that could break translation would be
// worse than no cache at all.
func cachedTranslation(prompt string) (string, bool) {
	dir, err := translationCacheDir()
	if err != nil {
		return "", false
	}
	b, err := os.ReadFile(filepath.Join(dir, translationKey(prompt)))
	if err != nil {
		return "", false
	}
	q := strings.TrimSpace(string(b))
	return q, q != ""
}

// storeTranslation records a successful translation.
//
// One file per entry, not one file holding all of them. The maintainer runs several
// projects at once, so a single JSON file would be read-modify-written by concurrent
// hooks and the loser's entry would vanish with no error — which is the failure the
// durable queue was given a directory to avoid. A file per key has no such race, and
// the write is atomic through a rename so a killed process cannot leave a half-written
// answer to be served as a real one.
//
// Only SUCCESSES are cached. A failure is a fact about the moment — an exhausted quota,
// a timeout — and caching it would turn a transient condition into a permanent one.
//
// No eviction, stated rather than implied. An entry is a query and its translation,
// on the order of a couple of hundred bytes; a year of heavy use is a few megabytes.
// Writing a retention policy that nothing implements would be worse than saying there
// is none.
func storeTranslation(prompt, query string) {
	dir, err := translationCacheDir()
	if err != nil || strings.TrimSpace(query) == "" {
		return
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return
	}
	final := filepath.Join(dir, translationKey(prompt))
	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return
	}
	name := tmp.Name()
	if _, err := tmp.WriteString(query); err != nil {
		tmp.Close()
		os.Remove(name)
		return
	}
	if err := tmp.Close(); err != nil {
		os.Remove(name)
		return
	}
	if err := os.Rename(name, final); err != nil {
		os.Remove(name)
	}
}

// translateFailure names WHY a translation did not run.
//
// It exists because the counter it replaces could not tell four conditions apart and
// printed a guess for all of them. The guess was "credential expired?", the condition
// was an exhausted weekly quota, and the diagnosis went to the wrong place — the
// credential was read, probed and found perfectly healthy before anyone thought to
// read the child's own error text, which had said so all along.
//
// Four conditions, four remedies: buy or wait for quota, re-authenticate, raise the
// timeout, install the binary. One number cannot carry that.
type translateFailure string

const (
	// failTimeout — the deadline fired. STRUCTURAL: read from the context, not from
	// any message, so it holds whatever language the child speaks.
	failTimeout translateFailure = "timeout"
	// failNoBinary — `claude` is not on PATH. Also structural.
	failNoBinary translateFailure = "no-binary"
	// failQuota — the usage limit is exhausted. Text-matched; see the note below.
	failQuota translateFailure = "quota"
	// failAuth — the credential was refused. Text-matched.
	failAuth translateFailure = "auth"
	// failUnclassified — it failed and this could not say why.
	//
	// A real bucket rather than a default that guesses. The whole defect being fixed
	// here was a guess printed with the confidence of a measurement, so an unreadable
	// failure must say it is unreadable: an honest "unclassified" sends the reader to
	// the evidence, and a wrong label sends them somewhere else entirely.
	failUnclassified translateFailure = "unclassified"
)

// classifyTranslateFailure says why the translator did not run.
//
// # Structural first, text second, and the order is the point
//
// A timeout and a missing binary are readable from Go's own values and cannot be
// wrong. Quota and auth are only distinguishable from the child's PROSE, and this
// project has measured what that costs: a message match is a claim about a program's
// human-readable output, and human-readable output is localised. `git` answers in
// Turkish on this machine, which is how a fail-closed guard came to refuse every
// init it was written to permit.
//
// So the text arm is deliberately the LAST resort, it is matched case-insensitively
// on several phrasings, and when nothing matches it returns unclassified rather than
// picking the likeliest. That last clause is the whole correction: the code being
// replaced picked the likeliest and was wrong.
func classifyTranslateFailure(ctx context.Context, err error, output string) translateFailure {
	// The deadline first. It is checked before anything else because a killed
	// subprocess also prints whatever it had got to, and that text would otherwise be
	// classified as though the process had finished and reported it.
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return failTimeout
	}
	if errors.Is(err, exec.ErrNotFound) {
		return failNoBinary
	}
	low := strings.ToLower(output)
	for _, m := range []string{"weekly limit", "usage limit", "rate limit", "quota", "limit reached"} {
		if strings.Contains(low, m) {
			return failQuota
		}
	}
	for _, m := range []string{"not logged in", "expired", "unauthorized", "invalid api key", "401"} {
		if strings.Contains(low, m) {
			return failAuth
		}
	}
	return failUnclassified
}

type translateOutcome int

const (
	translateOK      translateOutcome = iota // a usable query came back
	translateSkipped                         // nothing ran, or its answer was rejected
	translateFailed                          // the subprocess did not run to a usable exit
	translateCached                          // served from the cache; no subprocess, no quota
)

// translateCommand builds the translator subprocess. Separated from
// translatePrompt so the marking below is asserted by a test rather than only by
// reading — running the real binary is not an option in a unit test, and a
// mutation that dropped the marking left every other test green.
func translateCommand(ctx context.Context, prompt string, cred translatorCred) *exec.Cmd {
	cmd := exec.CommandContext(ctx, "claude", "-p", translationInstruction(prompt))
	cmd.Stdin = strings.NewReader("") // never inherit the hook's stdin
	// The child is a real Claude session, so it fires the SAME UserPromptSubmit
	// hook we are running inside — `atl retrieve` would search again (against this
	// instruction text) and `atl tick` would sweep for markers. Neither has a human
	// reader here. Left unmarked it costs a second full hybrid search on exactly
	// the prompts this feature exists to serve, and writes a phantom fire whose
	// page refs inflate the offer count the follow-through metric divides by.
	env := append(os.Environ(), atlInternalSessionEnv+"=1")
	// The credential is appended even when it came FROM the environment. Go keeps
	// the last occurrence of a duplicated key, so re-appending an inherited value
	// is a no-op — and one code path is worth more here than the branch saved,
	// because the branch that matters is the one where the value came from a file
	// and os.Environ() does not contain it at all. Without this line the resolver
	// would report a credential the child can never see: every non-English prompt
	// would pay a process spawn and up to the timeout, then fail, on exactly the
	// prompts the feature exists to serve.
	if cred.EnvName != "" {
		env = append(env, cred.EnvName+"="+cred.Value)
	}
	cmd.Env = env
	return cmd
}

// translationInstruction builds the prompt sent to the translator.
//
// Two things it deliberately asks for. Technical identifiers stay verbatim:
// `team.json`, `atl install`, a branch name or a symbol are already English and
// are precisely what BM25 matches best — translating them would destroy the
// signal the whole exercise exists to recover. And the output is a search query
// rather than a sentence, because BM25 scores terms, not grammar.
func translationInstruction(prompt string) string {
	return "Rewrite the text below as an English search query for a technical knowledge base.\n\n" +
		"Rules:\n" +
		"- Keep every technical identifier EXACTLY as written: file names, commands, flags, code symbols, branch names, issue references.\n" +
		"- Translate only the natural-language words.\n" +
		"- Output the query terms only — no explanation, no quotes, no preamble.\n" +
		"- If the text is already English, output it unchanged.\n\n" +
		"Text:\n" + prompt
}

// cleanTranslation validates what came back. A translator is an LLM and can
// answer the question instead of translating it, refuse, or explain itself; each
// of those is worse as a query than the original, so anything that does not look
// like a short query is rejected rather than used.
func cleanTranslation(raw, original string) (string, bool) {
	s := strings.TrimSpace(raw)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = strings.TrimSpace(s[:i]) // a query is one line; prose is not
	}
	s = strings.Trim(s, `"'`)
	if s == "" {
		return "", false
	}
	// A query is short; a refusal or an explanation is a sentence. Word count
	// separates those far more reliably than character length does — a polite
	// refusal can be shorter than a generous character bound while still being
	// prose ("I cannot translate this text because…" measured 158 chars against a
	// 187-char limit, and passed).
	//
	// Bound: twice the original's words, with a floor of 15 so a two-word query
	// is not held to four. Translation between languages is roughly
	// word-preserving; a 4x expansion is the model talking, not translating.
	words := len(strings.Fields(s))
	limit := 2 * len(strings.Fields(original))
	if limit < 15 {
		limit = 15
	}
	if words > limit {
		return "", false
	}
	if strings.EqualFold(s, strings.TrimSpace(original)) {
		return "", false // already English — nothing gained, skip the second search
	}
	return s, true
}

// atlInternalSessionEnv marks a `claude -p` that ATL itself started for a
// mechanical purpose, so the hook commands that run inside it can tell they have
// no human reader and do nothing.
//
// It is deliberately NOT set on the delivery engine's workers: those are real
// sessions doing real work, and retrieval and capture belong there.
const atlInternalSessionEnv = "ATL_INTERNAL_SESSION"

// isATLInternalSession reports whether this process is running inside such a
// session. Any hook body that would otherwise recurse or write a phantom event
// checks it first.
func isATLInternalSession() bool { return os.Getenv(atlInternalSessionEnv) != "" }

// claudeAvailable reports whether the translator can run at all: the binary
// exists and a credential is present. It does NOT spend an API call to find out
// — the hook fires on every prompt and must not pay for a liveness probe.
//
// A variable rather than a plain function so a test can state the condition it
// means. It probes the SYSTEM (a binary on PATH), which makes the developer's
// machine a richer environment than CI: locally `claude` is installed and a test
// that assumes so passes, while on a bare runner it cannot be true no matter
// what the environment says. That difference cost a red CI run.
var claudeAvailable = func() bool {
	if _, err := exec.LookPath("claude"); err != nil {
		return false
	}
	_, ok := translatorCredential()
	return ok
}

// translatorCredFile is the file a user may put the credential in, under the
// global layer (~/.atl). A plain file holding one opaque value, not JSON: it is
// hand-written, and wrapping a secret in a quoted format adds a paste hazard for
// no benefit at one value. A second credential later gets a second file, the way
// installed/<handle>__<name>.json partitions rather than nesting.
const translatorCredFile = "claude-token"

// translatorCredSourceFile is how the file names itself in a notice. Written
// with ~ rather than the resolved path so the message reads the same on every
// machine and carries no home directory into anyone's terminal.
const translatorCredSourceFile = "~/.atl/" + translatorCredFile

// translatorCred is the credential the translator needs AND the name of the
// variable it has to arrive in. The pair travels together because the child
// reads its credential from the environment: resolving one without carrying the
// other is what makes an available-but-unusable credential possible.
type translatorCred struct {
	EnvName string // the variable the subprocess must see
	Value   string // the secret itself
	Source  string // how to name it to the user, for the expired notice
	FromEnv bool   // whether it was already in this process's environment
}

// translatorCredentialPath is where the file lives. scope.LayerDir is the
// established accessor for the global layer rather than a tenth hand-built
// filepath.Join(home, ".atl", ...).
func translatorCredentialPath() (string, error) {
	dir, err := scope.LayerDir(scope.Global, "")
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, translatorCredFile), nil
}

// translatorCredential resolves the credential: environment first, then the
// file.
//
// The file exists because the environment cannot always be reached. A session's
// own login is held by the host and withheld from the processes it starts —
// measured on macOS as exactly five variables stripped between the host and its
// children, this credential among them — so ~/.claude/settings.json's env block,
// the documented place to set a variable, cannot deliver THIS variable to a
// hook. ~/.zshrc cannot either: zsh reads it only for interactive shells, and a
// hook is not one. That left ~/.zshenv as the single working export, which is a
// lot of per-machine shell knowledge to require for a feature that is meant to
// be optional and easy.
//
// Environment wins on purpose. It is the per-invocation, more specific channel,
// and silently preferring a file over a value the user explicitly exported would
// be the more surprising of the two orders.
//
// Every file failure — unresolvable path, unreadable, empty after trimming — is
// silence, never an error. This decides whether a subprocess runs and whether a
// notice prints, and the whole feature is fail-open: being wrong here must cost
// nothing. Empty is treated as absent rather than present so that a file created
// but never filled in produces the actionable "no credential" notice instead of,
// three prompts later, "your credential expired" about one the user never had.
func translatorCredential() (translatorCred, bool) {
	for _, name := range []string{"CLAUDE_CODE_OAUTH_TOKEN", "ANTHROPIC_API_KEY"} {
		if v := os.Getenv(name); v != "" {
			return translatorCred{EnvName: name, Value: v, Source: name, FromEnv: true}, true
		}
	}
	p, err := translatorCredentialPath()
	if err != nil {
		return translatorCred{}, false
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return translatorCred{}, false
	}
	v := strings.TrimSpace(string(b))
	if v == "" {
		return translatorCred{}, false
	}
	// A file-borne value is always presented as the OAuth token: there is no
	// sniffing by prefix. If the user pastes an API key here it will be rejected
	// by `claude`, and the expired notice then names this file — a self-describing
	// dead end, which is better than a brittle guess at which kind it is.
	return translatorCred{EnvName: "CLAUDE_CODE_OAUTH_TOKEN", Value: v, Source: translatorCredSourceFile}, true
}

// translationNotice is the session-start message shown when the corpus could
// benefit from translation but no credential is configured.
//
// Non-blocking by decision: it informs every session rather than gating anything,
// so it cannot be silently skipped and cannot stop work. The session's own
// identity is held by the host SDK and does not reach a child process — measured
// as 401 both with the inherited environment and with the auth vars stripped —
// so a separate token is genuinely required, not a convenience.
func translationNotice() string {
	return "atl: retrieval can translate a non-English prompt before searching — your knowledge base is English, so\n" +
		"     without translation such a prompt searches on one arm instead of two (measured: 25% vs 75% recall).\n" +
		"     It needs its own credential: a session's login is held by the host and is not visible to the\n" +
		"     tools it starts. Two places that do NOT reach it — ~/.claude/settings.json's env block (the host\n" +
		"     strips this variable before a hook runs) and ~/.zshrc (interactive shells only).\n" +
		"     Run `claude setup-token`, then put the token it prints in EITHER of these:\n" +
		"       " + translatorCredSourceFile + "          a plain file holding just the token (chmod 600 it)\n" +
		"       CLAUDE_CODE_OAUTH_TOKEN      exported from ~/.zshenv, which every shell reads\n" +
		"     Everything works without it — this is an improvement, not a requirement."
}

// retrievalTranslationNotice returns the session-start notice, and whether to
// show it at all.
//
// Shown only when it would change something: this project has an index (so
// retrieval is live here) and no credential is configured. A project with no
// corpus has nothing to translate against, and a configured machine has nothing
// to be told — silence in both cases keeps the notice meaningful when it does
// appear, which is the whole difference between a signal and noise.
func retrievalTranslationNotice(project string) (string, bool) {
	if project == "" {
		return "", false
	}
	idxPath, err := indexPathFor(project)
	if err != nil {
		return "", false
	}
	if _, err := os.Stat(idxPath); err != nil {
		return "", false // no index here — retrieval is not running in this project
	}
	if !claudeAvailable() {
		return translationNotice(), true
	}
	// A credential can be PRESENT and dead. `claudeAvailable` reads the env var and
	// deliberately spends no API call to check it — the hook fires on every prompt
	// and must not pay for a liveness probe. So expiry is detected the only way that
	// costs nothing: from the record of what actually happened.
	if translationFailing(idxPath) {
		// Resolved again rather than threaded down: claudeAvailable above proved one
		// exists, and the notice has to name WHICH source so the remedy points at the
		// place the user actually has to edit.
		cred, ok := translatorCredential()
		if !ok {
			return "", false
		}
		return expiredTranslationNotice(cred.Source), true
	}
	return "", false
}

// translationFailing reports whether recent translation attempts have all failed
// — the signature of a credential that is set but no longer valid.
//
// Three consecutive failures, not one: a single failure is a timeout or a blip,
// and a notice that fires on those is the constant channel this project has
// measured. Reads only the tail, and any read error means silence: this decides
// whether to PRINT something, so being wrong must cost nothing.
func translationFailing(idxPath string) bool {
	b, err := os.ReadFile(filepath.Join(filepath.Dir(idxPath), "fires.log"))
	if err != nil {
		return false
	}
	lines := strings.Split(strings.TrimRight(string(b), "\n"), "\n")
	seen := 0
	for i := len(lines) - 1; i >= 0 && seen < 3; i-- {
		f := strings.Split(lines[i], "\t")
		if len(f) < 2 {
			continue
		}
		switch f[1] {
		case "translated":
			return false // the most recent outcome is a success
		case "translate-failed":
			seen++
		}
	}
	return seen >= 3
}

// expiredTranslationNotice is the credential-is-set-but-dead variant. It says
// something different from the missing-credential notice on purpose: a user who
// has already run `setup-token` and exported the value reads "you need a
// credential" as wrong and stops reading.
// The remedy line is per-source, because the two sources are repaired in
// different places and sending a file-configured user to an environment variable
// (or the reverse) is the same drift this whole change exists to end.
func expiredTranslationNotice(source string) string {
	// "failed to run", not "all failed": after the outcome split, the recorded
	// token means only that the subprocess never reached a usable exit. The
	// sentence should say what was actually observed.
	body := "atl: retrieval translation is configured but its last three attempts failed to run —\n" +
		"     the credential is probably expired. A non-English prompt is searching on one arm\n" +
		"     instead of two (measured: 25% vs 75% recall), silently, because translation is\n" +
		"     fail-open by design.\n"
	if source == translatorCredSourceFile {
		return body + "     Re-run `claude setup-token` and replace the token in " + translatorCredSourceFile + "."
	}
	return body + "     Re-run `claude setup-token` and re-export " + source + " from ~/.zshenv\n" +
		"     (a hook sees neither ~/.zshrc nor ~/.claude/settings.json)."
}

// translatorCredentialCheck tightens the credential file's permissions when they
// are wider than owner-only.
//
// It earns its place precisely BECAUSE atl never writes this file. A
// hand-created one gets the user's umask — 0644 on a default macOS shell — and
// nothing else in the tree would ever notice. That makes this the one condition
// here that is mechanical, silent, and deterministically repairable, which is
// the whole test for belonging in the doctor rather than in a notice.
//
// It checks the mode and nothing else. It deliberately does NOT report "no
// credential configured", nor an empty file: the session-start notice already
// owns both, and a second channel for one condition is the always-fires failure
// this project has measured. So it is capable of silence in the ordinary case,
// and after healing once it goes quiet for good.
//
// The directory is left alone on purpose: ~/.atl is shared with the learning
// queue, which creates it 0o755, and tightening a directory this check does not
// own is a wider claim than "the file I read should not be world-readable".
func translatorCredentialCheck() doctor.Check {
	return func() doctor.Result {
		const name = "credential-file"
		const absent = "no translator credential file (optional)"
		p, err := translatorCredentialPath()
		if err != nil {
			return doctor.Result{Name: name, Status: doctor.OK, Detail: absent}
		}
		fi, err := os.Stat(p)
		if err != nil || fi.IsDir() {
			return doctor.Result{Name: name, Status: doctor.OK, Detail: absent}
		}
		mode := fi.Mode().Perm()
		if mode&0o077 == 0 {
			return doctor.Result{Name: name, Status: doctor.OK, Detail: translatorCredSourceFile + " is owner-only"}
		}
		if err := os.Chmod(p, 0o600); err != nil {
			return doctor.Result{Name: name, Status: doctor.Warn,
				Detail: fmt.Sprintf("%s is mode %04o and could not be tightened: %v", translatorCredSourceFile, mode, err)}
		}
		return doctor.Result{Name: name, Status: doctor.OK, Healed: true,
			Detail: fmt.Sprintf("%s was mode %04o — tightened to 0600", translatorCredSourceFile, mode)}
	}
}
