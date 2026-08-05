package commands

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"strings"
	"time"
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
	// translateTimeout bounds the translation subprocess. Generous on purpose:
	// the decision explicitly traded latency for recall, and a translation that
	// times out costs nothing but the original query, which is what would have
	// run anyway.
	translateTimeout = 20 * time.Second

	// maxTranslatableRunes caps what is sent. A retrieval query is a question,
	// not a document; something far longer is a paste, where the lexical arm has
	// plenty to match on already and translation buys nothing.
	maxTranslatableRunes = 600
)

// translatePrompt asks the user's own Claude for an English search query.
//
// It returns ok=false for every failure — no credential, a timeout, an empty or
// suspicious answer — and the caller then searches with the original prompt. The
// hook is fail-open by contract: translation may improve retrieval and must
// never be able to make it worse than not having tried.
//
// `claude -p` is invoked with NO --model flag on purpose. The user's own default
// model is used, so no model name exists in this codebase to go stale when a
// model is retired — a requirement of the decision, not an omission.
func translatePrompt(ctx context.Context, prompt string) (string, bool) {
	if len([]rune(prompt)) > maxTranslatableRunes {
		return "", false
	}
	ctx, cancel := context.WithTimeout(ctx, translateTimeout)
	defer cancel()

	cmd := translateCommand(ctx, prompt)
	var out, discard bytes.Buffer
	cmd.Stdout = &out
	// stderr is captured and dropped on purpose: this runs inside a hook whose
	// stdout is injected into the agent's context, and a stray warning from the
	// child would land there as if it were retrieved knowledge.
	cmd.Stderr = &discard
	if err := cmd.Run(); err != nil {
		return "", false
	}
	return cleanTranslation(out.String(), prompt)
}

// translateCommand builds the translator subprocess. Separated from
// translatePrompt so the marking below is asserted by a test rather than only by
// reading — running the real binary is not an option in a unit test, and a
// mutation that dropped the marking left every other test green.
func translateCommand(ctx context.Context, prompt string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, "claude", "-p", translationInstruction(prompt))
	cmd.Stdin = strings.NewReader("") // never inherit the hook's stdin
	// The child is a real Claude session, so it fires the SAME UserPromptSubmit
	// hook we are running inside — `atl retrieve` would search again (against this
	// instruction text) and `atl tick` would sweep for markers. Neither has a human
	// reader here. Left unmarked it costs a second full hybrid search on exactly
	// the prompts this feature exists to serve, and writes a phantom fire whose
	// page refs inflate the offer count the follow-through metric divides by.
	cmd.Env = append(os.Environ(), atlInternalSessionEnv+"=1")
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
	return os.Getenv("CLAUDE_CODE_OAUTH_TOKEN") != "" || os.Getenv("ANTHROPIC_API_KEY") != ""
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
		"     This needs its own credential: a session's login is not visible to the tools it starts.\n" +
		"     To enable it, run `claude setup-token` and export the value as CLAUDE_CODE_OAUTH_TOKEN.\n" +
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
	if project == "" || claudeAvailable() {
		return "", false
	}
	idxPath, err := indexPathFor(project)
	if err != nil {
		return "", false
	}
	if _, err := os.Stat(idxPath); err != nil {
		return "", false // no index here — retrieval is not running in this project
	}
	return translationNotice(), true
}
