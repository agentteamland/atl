package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/agentteamland/atl/cli/internal/detach"
	"github.com/agentteamland/atl/cli/internal/retrieve"
	"github.com/agentteamland/atl/cli/internal/scope"
	"github.com/agentteamland/atl/cli/internal/throttle"
	"github.com/agentteamland/atl/cli/internal/transcript"
	"github.com/spf13/cobra"
)

// envNoAutoIndex opts out of the background index refresh (a safety valve for the
// one-time full build's cost), matching the ATL_NO_SELF_UPDATE / ATL_NO_TEAM_UPDATE
// pattern.
const envNoAutoIndex = "ATL_NO_RETRIEVE_INDEX"

// retrieveAutoIndexThrottle bounds how often session-start fires a rebuild.
//
// It does NOT prevent overlapping builds, and used to be relied on for that: the
// window was set above the then-current cold-build time so a restart mid-build
// could not spawn a second. That assumption broke silently when the corpus grew
// — a build now outlasts the window, so a session-start behind it sees a corpus
// still stale and spawns another. retrieve.AcquireBuildLock is what actually
// guarantees single-flight; this only keeps the cheap checks from running on
// every session.
const retrieveAutoIndexThrottle = 10 * time.Minute

// retrieveTopK is the MAXIMUM number of knowledge pages the hook surfaces. It is
// a ceiling, not a quota — since the semantic arm gained a floor, a fire may
// legitimately produce fewer, including none.
//
// Deliberately left at 5 rather than tightened: the one refuting page in the
// measured failure set had a best English rank of 6 of 182, so a smaller ceiling
// would exclude it even after the ranker is repaired.
const retrieveTopK = 5

// minPromptRunes is the length below which a prompt is not ranked at all.
//
// Counted in RUNES, not bytes. An accented or non-Latin script costs 2-3 bytes
// per character, so a byte floor admits a short prompt in one language while
// suppressing an equally short one in another — it would filter unevenly by
// language, and this corpus is read in two. Set at 12: above one-word
// acknowledgements, below any real question.
const minPromptRunes = 12

// retrieveInput is the UserPromptSubmit hook payload atl retrieve reads on stdin.
type retrieveInput struct {
	Prompt string `json:"prompt"`
	CWD    string `json:"cwd"`
}

var retrieveCmd = &cobra.Command{
	Use:   "retrieve",
	Short: "Per-prompt knowledge retrieval (hybrid lexical + semantic)",
	// No longer Hidden. It was, correctly, while this was purely a hook body that
	// nobody typed — but `--query` (the agent-initiated consult) and `stats` are
	// deliberately invoked now, and the repo's own convention is that a hidden
	// command is unhidden in the same change that makes it user-facing.
	Long: "The read side of ATL's knowledge loop (the write side is learning-capture +\n" +
		"/drain). Run with no subcommand as a UserPromptSubmit hook: it reads the prompt\n" +
		"on stdin, ranks the project's knowledge pages (BM25 + a local semantic embedder,\n" +
		"RRF-fused), and prints the top matches as context. Fail-open — any error prints\n" +
		"nothing and never blocks the prompt.\n\n" +
		"  atl retrieve --query <terms>   search with a query you wrote yourself\n" +
		"  atl retrieve index             (re)build the index for the current project\n" +
		"  atl retrieve warm              download the embedding model and warm the pipeline",
	SilenceUsage: true,
	// The bare `atl retrieve` is the hook body; --query is the agent-initiated path.
	RunE: func(cmd *cobra.Command, args []string) error {
		if q, _ := cmd.Flags().GetString("query"); strings.TrimSpace(q) != "" {
			return runConsult(cmd, q)
		}
		runRetrieveHook(cmd)
		return nil // fail-open: never surface an error that would break the prompt
	},
}

// runConsult is the agent-initiated read: the model wrote the query itself,
// rather than the hook embedding whatever sentence the user happened to type.
//
// Measured on `.atl/eval/retrieval-answer-key.json` (12 questions, both
// languages, the real Query path): the raw prompt scores 83% English / 75%
// Turkish at recall@5, a model-written query scores 100%, and English and
// Turkish become IDENTICAL because the query is born in the corpus's language.
//
// Unlike the hook this is NOT silent on failure — the agent asked, so it gets an
// answer or a reason. Fail-open is a property of something that fires
// uninvited; a tool the agent chose to call should say when it found nothing.
func runConsult(cmd *cobra.Command, query string) error {
	root, err := projectKey()
	if err != nil {
		return err
	}
	results, err := queryIndex(cmd.Context(), root, query)
	if err != nil {
		return err
	}
	if len(results) == 0 {
		// Logged, because "the agent asked and the corpus had nothing" is a
		// different and more interesting event than "the agent never asked".
		logRetrieveFire(root, "consulted-empty", nil)
		fmt.Fprintf(cmd.OutOrStdout(), "No page in this project's knowledge base matched %q.\n", query)
		return nil
	}
	paths := make([]string, len(results))
	for i, r := range results {
		paths[i] = r.Path
	}
	logRetrieveFire(root, "consulted", paths)
	fmt.Fprint(cmd.OutOrStdout(), formatResultsWithHeader(root, results,
		fmt.Sprintf("[atl] %d page(s) for your query %q — open the path for the full page.\n", len(results), query)))
	return nil
}

// queryIndex is the search itself, shared by the hook and the agent-initiated
// path so the two can never diverge on ranking, floors or topK — a consult that
// scored differently from the hook would make every comparison between them
// meaningless.
func queryIndex(parent context.Context, root, query string) ([]retrieve.Result, error) {
	idxPath, err := indexPathFor(root)
	if err != nil {
		return nil, err
	}
	ix, err := retrieve.Load(idxPath)
	if err != nil || ix == nil {
		return nil, fmt.Errorf("no retrieval index for this project yet — run `atl retrieve index`")
	}
	ctx, cancel := context.WithTimeout(parent, 30*time.Second)
	defer cancel()
	e := embedderIfModelPresent(ctx)
	if e != nil {
		defer e.Close()
	}
	return ix.Query(ctx, query, e, retrieveTopK, retrieve.MinSimDefault)
}

// runRetrieveHook is the UserPromptSubmit hook. It is exhaustively fail-open:
// every failure path simply prints nothing, so a missing index, an absent model,
// or a parse error can never block or corrupt the prompt. The deferred recover is
// part of that contract — this hook fires on every prompt, and a panic deep in
// the embedder would exit non-zero and erase the prompt; nothing is printed until
// the single final write, so recovering here cleanly drops the retrieval instead.
func runRetrieveHook(cmd *cobra.Command) {
	defer func() { _ = recover() }()
	if isATLInternalSession() {
		return
	}

	data, err := io.ReadAll(cmd.InOrStdin())
	if err != nil {
		return
	}
	var in retrieveInput
	if err := json.Unmarshal(data, &in); err != nil {
		return
	}
	prompt := strings.TrimSpace(in.Prompt)
	if prompt == "" {
		return
	}
	// The payload's directory, resolved to the repository that owns it. The turn
	// hook resolves the same way, which is what keeps one session's fires and its
	// turns in one bucket instead of splitting them wherever the agent cd'd to.
	root := in.CWD
	if root == "" {
		root, _ = os.Getwd()
	}
	root = projectKeyFrom(root)

	// Two suppressions before any work. Measured over one real session: of 277
	// fires, 112 ranked machine-injected notification text and 43 ranked a prompt
	// of a few characters — 155 (56%) carried no question for the corpus to answer.
	// Ranking those is not merely wasted; it is most of the volume that taught the
	// reader to skip the channel.
	if strings.HasPrefix(prompt, "<") {
		logRetrieveFire(root, "suppressed-machine", nil)
		return // <task-notification>, <system-reminder>, and friends
	}
	if len([]rune(prompt)) < minPromptRunes {
		logRetrieveFire(root, "suppressed-short", nil)
		return
	}

	idxPath, err := indexPathFor(root)
	if err != nil {
		return
	}
	ix, err := retrieve.Load(idxPath)
	if err != nil || ix == nil {
		return // no index yet (built on drain) — nothing to inject
	}

	// Translate before searching when the query cannot reach this corpus lexically.
	//
	// The trigger is the lexical arm returning NOTHING — not a language check.
	// That is deliberate: it is language-agnostic, it costs nothing on an English
	// query (which always has lexical hits on a topic the corpus covers), and it
	// fires precisely in the case translation exists for. Retrieval on a
	// non-English prompt otherwise runs on the semantic arm alone, which measures
	// 25% against 75% for the same questions in English.
	//
	// The budget is separate from the 5s search budget below, because this is the
	// one part the decision explicitly agreed to pay time for.
	query := prompt
	if ix.LexicalHits(prompt) == 0 && claudeAvailable() {
		translated, outcome, why := translatePrompt(cmd.Context(), prompt)
		switch outcome {
		case translateOK:
			query = translated
			logRetrieveFire(root, "translated", nil)
		case translateCached:
			// Logged under its own token so the saving is MEASURABLE. The card that
			// produced this work could not size the cache from the existing record,
			// because the log kept the outcome and never the input — there was
			// nothing to deduplicate against. From here the rate measures itself.
			query = translated
			logRetrieveFire(root, "translated:cached", nil)
		case translateFailed:
			// Record the failure too. Translation is fail-open and silent by
			// design, so without this line a translator that has STOPPED WORKING is
			// indistinguishable from one that is working: the credential is still
			// there, the notice stays suppressed, and every non-English prompt
			// quietly searches on one arm. This is the only evidence of that
			// state that costs nothing to collect.
			//
			// The REASON rides in the token. The old form logged a bare
			// "translate-failed" for four different conditions, so the stats line
			// could only guess at the cause — and its guess ("credential expired?")
			// was wrong about the one that was actually happening. An old line with
			// no reason still parses; it counts as unclassified, which is honest,
			// because nothing recorded what those failures were.
			logRetrieveFire(root, "translate-failed:"+string(why), nil)
		default:
			// The subprocess ran and its answer was not used — already English,
			// or prose where a query was asked for. Logged under its own token so
			// it stays out of the expired-credential evidence: three ordinary
			// English prompts must never add up to "your credential is dead".
			logRetrieveFire(root, "translate-skipped", nil)
		}
	}

	// The embedder is best-effort: if the model isn't downloaded or fails to load,
	// query with a nil embedder (BM25-only) rather than skipping retrieval entirely.
	ctx, cancel := context.WithTimeout(cmd.Context(), 5*time.Second)
	defer cancel()
	e := embedderIfModelPresent(ctx)
	if e != nil {
		defer e.Close()
	}

	results, err := ix.Query(ctx, query, e, retrieveTopK, retrieve.MinSimDefault)
	if err != nil {
		return
	}
	if len(results) == 0 {
		logRetrieveFire(root, "silent", nil)
		return // nothing cleared the floor — say nothing, which is now a real outcome
	}
	paths := make([]string, len(results))
	for i, r := range results {
		paths[i] = r.Path
	}
	logRetrieveFire(root, "fired", paths)
	fmt.Fprint(cmd.OutOrStdout(), formatResults(root, results))
}

// embedderIfModelPresent returns a ready embedder only if the model is already
// downloaded — the hook must never trigger a multi-second download on a prompt.
// nil means "run BM25-only", not an error.
func embedderIfModelPresent(ctx context.Context) *retrieve.Embedder {
	dir, ok := retrieve.ModelDirIfPresent()
	if !ok {
		return nil
	}
	e, err := retrieve.NewEmbedder(ctx, dir)
	if err != nil {
		return nil
	}
	return e
}

var retrieveIndexCmd = &cobra.Command{
	Use:          "index",
	Short:        "(Re)build the retrieval index for the current project",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		timeout, _ := cmd.Flags().GetDuration("timeout")
		ctx, cancel := context.WithTimeout(cmd.Context(), timeout)
		defer cancel()

		root, err := projectKey()
		if err != nil {
			return err
		}
		dirs, err := corpusDirs(root)
		if err != nil {
			return err
		}
		docs, err := retrieve.WalkCorpus(dirs)
		if err != nil {
			return err
		}
		out := cmd.OutOrStdout()
		if len(docs) == 0 {
			fmt.Fprintln(out, "atl retrieve: no knowledge pages found for this project")
			return nil
		}

		// The embedder is optional: --lexical, or a model that won't download/load,
		// yields a BM25-only index (retrieval still works, degraded to lexical)
		// rather than failing the build.
		lexical, _ := cmd.Flags().GetBool("lexical")
		var e *retrieve.Embedder
		mode := "lexical-only"
		if !lexical {
			if dir, derr := retrieve.EnsureModel(ctx); derr != nil {
				fmt.Fprintf(out, "atl retrieve: model unavailable (%v); building a lexical-only index\n", derr)
			} else if emb, eerr := retrieve.NewEmbedder(ctx, dir); eerr != nil {
				fmt.Fprintf(out, "atl retrieve: embedder unavailable (%v); building a lexical-only index\n", eerr)
			} else {
				e = emb
				defer e.Close()
				mode = "hybrid"
			}
		}

		idxPath, err := indexPathFor(root)
		if err != nil {
			return err
		}

		// Single-flight. A full build outlasts the auto-index throttle on a large
		// corpus, so without this a second spawn starts behind the first and both
		// saturate the machine — measured at 832% CPU with two running.
		release, ok := retrieve.AcquireBuildLock(idxPath)
		if !ok {
			fmt.Fprintln(out, "atl retrieve: another index build is already running for this project")
			return nil // a duplicate is not a failure
		}
		defer release()

		// Bound the build to half the machine. This is a background nicety that
		// nobody asked for at this moment: the pure-Go backend parallelises across
		// GOMAXPROCS, and left unbounded it takes every core and spins the fans on a
		// laptop whose owner is doing something else. Halving roughly doubles the
		// wall clock of a job already measured in minutes, which costs a background
		// job nothing and costs an interactive machine a great deal.
		if n := runtime.NumCPU() / 2; n >= 1 {
			defer runtime.GOMAXPROCS(runtime.GOMAXPROCS(n))
		}

		old, _ := retrieve.Load(idxPath) // reuse unchanged pages; nil on first build

		t0 := time.Now()
		ix := retrieve.BuildIncremental(ctx, docs, e, old)

		// A build that ran out of context must NOT be written. When the deadline
		// fires, every page the build had not reached carries a nil vector — and
		// those nils are permanent, because the incremental reuse key is
		// (path, text): a page saved without a vector matches its own checksum on
		// the next build and is reused as though it had been embedded, so it is
		// invisible to the semantic arm for good. The corruption looks FRESH to the
		// freshness check, which is why a retry repairs nothing.
		//
		// Refusing the write leaves the previous index in place. That is strictly
		// better than a fresh-looking corrupt one, and it makes a too-short deadline
		// a loud failure instead of a silent quality regression nobody can attribute.
		//
		// The check is on the CONTEXT and not on the shape of the vectors, and that
		// distinction is load-bearing: a nil vector is also the documented outcome
		// for a single document that could not be embedded ("best-effort ... stays
		// lexically searchable" — see BuildIncremental). Refusing every index that
		// contains a nil would break that on purpose-built behaviour. Only the
		// context separates "one page failed" from "the build was cut off".
		if cerr := ctx.Err(); cerr != nil {
			return fmt.Errorf("atl retrieve: the index build did not finish (%w) — %s left untouched; re-run with a longer --timeout", cerr, idxPath)
		}

		if err := ix.Save(idxPath); err != nil {
			return err
		}
		fmt.Fprintf(out, "atl retrieve: indexed %d pages (%s) in %.1fs → %s\n", len(docs), mode, time.Since(t0).Seconds(), idxPath)
		return nil
	},
}

// retrieveWarmCmd downloads + verifies the embedding model and runs one embed to
// prove the stack. A one-time prefetch and a way to validate the pipeline.
var retrieveWarmCmd = &cobra.Command{
	Use:          "warm",
	Short:        "Download the embedding model and warm the pipeline",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		timeout, _ := cmd.Flags().GetDuration("timeout")
		ctx, cancel := context.WithTimeout(cmd.Context(), timeout)
		defer cancel()
		out := cmd.OutOrStdout()

		t0 := time.Now()
		dir, err := retrieve.EnsureModel(ctx)
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "model ready: %s (%.1fs)\n", dir, time.Since(t0).Seconds())

		t1 := time.Now()
		emb, err := retrieve.NewEmbedder(ctx, dir)
		if err != nil {
			return err
		}
		defer emb.Close()
		vecs, err := emb.Embed(ctx, []string{"how does the dispatch merge-verify work"})
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "embedded ok: dim=%d cold=%dms\n", len(vecs[0]), time.Since(t1).Milliseconds())
		return nil
	},
}

// corpusDirs returns the knowledge-corpus roots to index for a project: its own
// knowledge — the wiki (current truth) and journal (history). A delivery project
// (GitHub backend) keeps durable knowledge in the in-repo docs/ tree, so docs/ is
// added ONLY when the .delivery marker is present — an ordinary repo's docs/ site
// is often a large vendored tree (node_modules and all) that must not pollute the
// corpus. The agent knowledge base is a deferred corpus expansion (global agent
// KBs need cross-project relevance gating); v1 surfaces project knowledge, the
// core of what #140 exists for.
//
// .atl/brain-storms is deliberately NOT here, and the reason is not size. A
// brainstorm records rejected options by mandate — 35 of this workspace's 52 do
// — and a chunk of one, split from the verdict that rejected it, reads exactly
// like a decision. That is the shape of the failure this whole change is about
// (a claim that a decision existed when it did not), so indexing the process
// layer would feed it. .atl/docs carries the same discussions' *outcomes* and is
// indexed; brainstorms stay a deliberate read.
func corpusDirs(projectRoot string) ([]string, error) {
	atlDir, err := scope.LayerDir(scope.Project, projectRoot)
	if err != nil {
		return nil, err
	}
	dirs := []string{
		filepath.Join(atlDir, "wiki"),
		filepath.Join(atlDir, "journal"),
		// Settled decisions. Current truth like the wiki, and the layer most likely
		// to refute a proposal — a measured failure was refuted by a line in
		// .atl/docs/atl-v2.md that retrieval could not reach.
		filepath.Join(atlDir, "docs"),
		// Installed team knowledge: what actually executes in this project. Two of
		// the six measured failures were refuted by files under here — one in
		// knowledge/, one in an agent's children/ — and neither was reachable.
		// Named subdirectories rather than .claude itself, so worktrees/, settings
		// and future non-knowledge additions stay out.
		filepath.Join(projectRoot, ".claude", "agents"),
		filepath.Join(projectRoot, ".claude", "knowledge"),
		filepath.Join(projectRoot, ".claude", "skills"),
		filepath.Join(projectRoot, ".claude", "backends"),
		filepath.Join(projectRoot, ".claude", "packs"),
	}
	// The team SOURCE tree, which exists only in the repo that owns it. The
	// installed copies above are what a CONSUMING project executes; this is what a
	// session EDITING a team is working on, and most sessions in that repo are
	// exactly that. The two are not the same question — an installed copy can lag
	// its source by a version, and in the owning repo the source is authoritative.
	//
	// Concretely: the two knowledge files that motivated the whole corpus-reach
	// investigation — a pack format and a decomposition blueprint — both live here,
	// each refuted a proposal that reached the user anyway, and neither was
	// retrievable from a session in this repo.
	//
	// Measured before adding it, because the card required both checks. SIZE: the
	// owning repo's corpus is 1 file today, so this is not a widening of a large
	// index but the difference between retrieval existing there and not; it costs
	// a consuming project nothing, since teams/ does not exist there.
	// DOUBLE-INDEXING: the concern was source and installed copies ranking twice
	// under two paths — but the owning repo has no installed teams at all (zero
	// manifests), so the two never coexist.
	if _, err := os.Stat(filepath.Join(projectRoot, "teams")); err == nil {
		dirs = append(dirs, filepath.Join(projectRoot, "teams"))
	}
	if _, err := os.Stat(filepath.Join(projectRoot, ".delivery", "config.json")); err == nil {
		dirs = append(dirs, filepath.Join(projectRoot, "docs"))
	}
	return dirs, nil
}

// indexPathFor is the on-disk index for a project: a global cache keyed by the
// project path, so it is never committed and each project (and git worktree) has
// its own — ~/.atl/cache/retrieve/<project-slug>/index.gob.
// logRetrieveFire appends one line per hook invocation next to the index. It is
// the subsystem's first denominator: before this, "how often does retrieval have
// anything to say, and does anyone read it?" could only be answered by a forensic
// pass over a 30 MB transcript.
//
// Strictly best-effort — every error is dropped. A hook that fails open on a
// missing model must not start failing on a full disk.
//
// Deliberately NOT a pruning signal. A page that is never surfaced is not thereby
// unwanted: in the measured failure set the one indexed refuter sits in the
// never-surfaced set, so a "delete what never appears" rule would have removed
// precisely the page whose absence was the failure. That bound holds while the
// ranker is known-broken for the majority language; re-derive it once it is not.
func logRetrieveFire(projectRoot, outcome string, paths []string) {
	idx, err := indexPathFor(projectRoot)
	if err != nil {
		return
	}
	line := time.Now().UTC().Format(time.RFC3339) + "\t" + outcome
	if len(paths) > 0 {
		line += "\t" + strings.Join(paths, " ")
	}
	dir := filepath.Dir(idx)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return
	}
	f, err := os.OpenFile(filepath.Join(dir, "fires.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = f.WriteString(line + "\n")
}

// retrieveFireStats is the parsed tally behind `atl retrieve stats`.
//
// Total counts PROMPTS, and every percentage this command prints is computed
// against it — so only an outcome that represents one prompt may increment it.
type retrieveFireStats struct {
	Total, Fired, Silent, Machine, Short int
	Translated                           int
	// TranslatedCached is the subset served without spawning anything. It is the
	// quota NOT spent, which is the number the user actually wants and the one the
	// previous log shape could never produce.
	TranslatedCached int
	TranslateFailed  int
	// TranslateFailedBy is the same total, split by WHY. The total alone could not
	// name a cause, so the line that rendered it guessed one — and guessed wrong
	// about the condition that was actually happening.
	TranslateFailedBy                map[string]int
	Turns, Consulted, ConsultedEmpty int
	Offered                          int
	Pages                            map[string]int
}

// readFireStats tallies the fire log. A missing log is an empty tally, not an
// error — the log only exists once the hook has run since this shipped.
// translateFailureOrder is the reason table the stats line renders from: the key as
// logged, and the remedy that key implies.
//
// A table rather than a switch, and the remedy sits BESIDE the key rather than being
// written at the call site, because the defect being fixed here was exactly a remedy
// that had come adrift from the condition it described. Four conditions with four
// different answers cannot share one sentence.
//
// Ordered by what a reader should act on first, not alphabetically.
var translateFailureOrder = []struct{ key, remedy string }{
	{"quota", "(the usage limit is exhausted — this spends the same weekly budget as a session)"},
	{"auth", "(the credential was refused — re-authenticate; see the session-start notice)"},
	{"timeout", "(the translator was still running when its deadline fired)"},
	{"no-binary", "(`claude` is not on PATH for the process the hook starts)"},
	{"unclassified", "(it failed and the log could not say why — lines written before reasons existed count here)"},
}

func readFireStats(logPath string) (retrieveFireStats, error) {
	st := retrieveFireStats{Pages: map[string]int{}}
	b, err := os.ReadFile(logPath)
	if err != nil {
		if os.IsNotExist(err) {
			return st, nil
		}
		return st, err
	}
	for _, line := range strings.Split(string(b), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		f := strings.Split(line, "\t")
		if len(f) < 2 {
			continue
		}
		// "translated" is written BEFORE the search and the search then writes its
		// own outcome, so a translated prompt produces two lines. It is a modifier
		// on the fire that follows, not a fire of its own: counting it would inflate
		// the denominator every percentage below is computed against — and inflate
		// it further the more the feature is used, so the instrument would degrade
		// exactly as the thing it measures gets adopted.
		if strings.HasPrefix(f[1], "translated") {
			st.Translated++
			if f[1] == "translated:cached" {
				st.TranslatedCached++
			}
			continue
		}
		// Same reasoning, other outcome: a skipped translation is a modifier on the
		// fire that follows, not a fire of its own. Left to fall through it would
		// land in st.Total and inflate the denominator every percentage divides by —
		// and inflate it most for English-speaking users, whose prompts are the ones
		// that get skipped.
		//
		// The same holds for BOTH other translate outcomes, and translate-failed was
		// the one already getting this wrong. Control falls through to the search
		// after every outcome, so a failed translation is followed by its own
		// fired/silent line exactly like a successful one — it is a modifier, not a
		// second prompt. Counting it inflated the denominator hardest on a machine
		// whose credential has EXPIRED, which is precisely the state the notice
		// exists to detect, so the instrument degraded exactly where it was needed.
		if f[1] == "translate-skipped" {
			continue
		}
		if strings.HasPrefix(f[1], "translate-failed") {
			st.TranslateFailed++
			// The reason rides after a colon. A line written before reasons existed
			// has none, and it counts as unclassified rather than being assigned to
			// the likeliest bucket — nothing recorded what those failures were, and
			// inventing a label for them here would be the same fabrication this
			// change exists to remove, one layer down.
			why := "unclassified"
			if _, r, found := strings.Cut(f[1], ":"); found && r != "" {
				why = r
			}
			if st.TranslateFailedBy == nil {
				st.TranslateFailedBy = map[string]int{}
			}
			st.TranslateFailedBy[why]++
			continue
		}
		// Turns and consults are not prompts either. A turn is the denominator
		// itself, and a consult is something the AGENT did — counting either as a
		// fire would inflate the number every percentage below divides by, the
		// exact defect the translated line above exists to prevent.
		switch f[1] {
		case "turn":
			st.Turns++
			continue
		case "consulted":
			st.Consulted++
			if len(f) > 2 {
				st.Offered += len(strings.Fields(f[2]))
				for _, p := range strings.Fields(f[2]) {
					st.Pages[p]++
				}
			}
			continue
		case "consulted-empty":
			st.ConsultedEmpty++
			continue
		}
		st.Total++
		switch f[1] {
		case "fired":
			st.Fired++
			if len(f) > 2 {
				for _, p := range strings.Fields(f[2]) {
					st.Pages[p]++
					st.Offered++
				}
			}
		case "silent":
			st.Silent++
		case "suppressed-machine":
			st.Machine++
		case "suppressed-short":
			st.Short++
		}
	}
	return st, nil
}

func indexPathFor(projectRoot string) (string, error) {
	layer, err := scope.LayerDir(scope.Global, "")
	if err != nil {
		return "", err
	}
	abs, err := filepath.Abs(projectRoot)
	if err != nil {
		abs = projectRoot
	}
	return filepath.Join(layer, "cache", "retrieve", transcript.SlugForPath(abs), "index.gob"), nil
}

// formatResults renders the retrieved pages as a plain-text context block for
// UserPromptSubmit (whose stdout is injected verbatim). Each entry is a pointer —
// title, project-relative path, and a short excerpt — so the agent knows which
// pages to open, without flooding the prompt with full page bodies.
func formatResults(projectRoot string, results []retrieve.Result) string {
	return formatResultsWithHeader(projectRoot, results,
		"[atl] Knowledge pages that may be relevant to this prompt (atl#140 retrieval).\n"+
			"Consult any that match the topic before answering; open the path for the full page.\n")
}

// formatResultsWithHeader exists because the two readers are in different
// positions. The hook speaks to an agent that did not ask and may not care, so
// it hedges ("may be relevant") and tells it what to do. A consult answers an
// agent that asked a specific question, so hedging is noise and the instruction
// is already obeyed — what it needs instead is its own query echoed back, since
// a wrong query is the most likely reason a consult returns the wrong pages.
func formatResultsWithHeader(projectRoot string, results []retrieve.Result, header string) string {
	var b strings.Builder
	b.WriteString(header)
	for i, r := range results {
		rel := r.Path
		if p, err := filepath.Rel(projectRoot, r.Path); err == nil && !strings.HasPrefix(p, "..") {
			rel = p
		}
		fmt.Fprintf(&b, "%d. %s — %s\n", i+1, r.Title, rel)
		if ex := excerpt(r.Path); ex != "" {
			fmt.Fprintf(&b, "   %s\n", ex)
		}
	}
	return b.String()
}

// excerpt reads a short, whitespace-collapsed lead from a page — skipping a YAML
// frontmatter block and Markdown headings — or "" if the page can't be read.
// Bounded so the injected block stays small.
func excerpt(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	lines := strings.Split(string(b), "\n")
	// Skip a leading `---` … `---` frontmatter block if present.
	if len(lines) > 0 && strings.TrimSpace(lines[0]) == "---" {
		for i := 1; i < len(lines); i++ {
			if strings.TrimSpace(lines[i]) == "---" {
				lines = lines[i+1:]
				break
			}
		}
	}
	var body []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		body = append(body, line)
		if len(body) >= 3 {
			break
		}
	}
	s := strings.Join(body, " ")
	const max = 220
	if len(s) > max {
		s = strings.TrimSpace(s[:max]) + "…"
	}
	return s
}

// autoIndexRetrieval refreshes the project's retrieval index in the background
// when its knowledge corpus has changed since the last build. It runs from
// session-start: deterministic (no reliance on the LLM drain skill), self-healing,
// and reaching any project a session opens — the "index on drain" mechanism.
// Fail-open: it never blocks or fails the hook. It skips git worktrees so
// `atl work dispatch`'s N per-worktree workers don't each storm a full rebuild
// (per-worker retrieval is a separate follow-up), and honors ATL_NO_RETRIEVE_INDEX.
func autoIndexRetrieval(project string) {
	if project == "" || os.Getenv(envNoAutoIndex) != "" {
		return
	}
	stamp, err := throttle.StampPath("retrieve-index-" + transcript.SlugForPath(project))
	if err != nil || !throttle.Gate(stamp, retrieveAutoIndexThrottle) {
		return
	}
	if inGitWorktree(project) { // the git exec runs only after the cheap gate passes
		return
	}
	dirs, err := corpusDirs(project)
	if err != nil {
		return
	}
	idxPath, err := indexPathFor(project)
	if err != nil {
		return
	}
	migrating := indexUnusable(idxPath)
	if !migrating && !corpusStale(dirs, idxPath) {
		return
	}
	_ = throttle.Touch(stamp)
	if migrating {
		announceMigration(dirs)
	}
	_ = detach.Spawn("retrieve", "index") // detached: inherits cwd (= project)
}

// indexUnusable reports whether an index FILE exists that this binary cannot
// read — the signature of an embedder swap (indexFormatVersion moved), and also
// of a corrupt cache.
//
// It exists because corpusStale cannot see this case: it compares modtimes, and
// an embedder swap makes no corpus file newer. Over a quiet corpus the index
// would stay "fresh" forever while Load reports "no index", so retrieval would
// go silently dead until something happened to touch a page. That gap was known
// and documented as pre-existing; a model swap is what makes it load-bearing.
func indexUnusable(idxPath string) bool {
	if _, err := os.Stat(idxPath); err != nil {
		return false // no index at all — the modtime pass already handles that
	}
	ix, err := retrieve.Load(idxPath)
	return err == nil && ix == nil
}

// announceMigration tells the user a full rebuild is starting and why.
//
// Not decoration: the rebuild is detached, invisible to the task panel, and on a
// large corpus runs for tens of minutes — the exact shape of the incident where
// two index builds saturated the machine and the only channel to the user was
// the fans. A one-time cost is acceptable; an unannounced one is not.
func announceMigration(dirs []string) {
	pages := 0
	for _, dir := range dirs {
		_ = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
			if err == nil && !d.IsDir() && strings.HasSuffix(p, ".md") {
				pages++
			}
			return nil
		})
	}
	fmt.Printf("atl: the embedding model changed — rebuilding the retrieval index from scratch "+
		"in the background (~%d pages). This is a one-time cost per project, bounded to half "+
		"your cores; set %s=1 to skip it.\n", pages, envNoAutoIndex)
}

// corpusStale reports whether any corpus Markdown file is newer than the index
// (or the index is missing), or the index holds a page that has since been
// deleted — and there is at least one file to index. A missing index with a
// non-empty corpus is stale (build it); an empty corpus never is (a rebuild would
// have nothing to index).
func corpusStale(dirs []string, idxPath string) bool {
	var idxTime time.Time
	if info, err := os.Stat(idxPath); err == nil {
		idxTime = info.ModTime()
	} // else zero time: any corpus file counts as newer
	var any, newer bool
	for _, dir := range dirs {
		_ = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(p, ".md") {
				return nil
			}
			any = true
			if info, e := d.Info(); e == nil && info.ModTime().After(idxTime) {
				newer = true
			}
			return nil
		})
	}
	return any && (newer || indexHasDeletedPage(idxPath))
}

// indexHasDeletedPage reports whether the index holds a page whose file is gone.
// A delete-only change makes nothing that survives newer, so modtimes alone never
// see it — and the deleted page goes on being ranked (with a stale title and, the
// only tell, no excerpt) while it evicts a live result from the fixed top-k
// budget. Checked only after the cheap modtime pass says otherwise-fresh.
func indexHasDeletedPage(idxPath string) bool {
	ix, err := retrieve.Load(idxPath)
	if err != nil || ix == nil {
		// Nothing to compare against. A *missing* index is already caught by the
		// modtime pass (nothing to stat -> zero time -> every page counts as
		// newer); a present-but-unusable one — corrupt, or written by an older
		// indexFormatVersion, both of which Load also reports as "no index" — is
		// not, so over a quiet corpus it stays fresh here and never rebuilds.
		// That gap predates this check; it is left as-is rather than widened in.
		return false
	}
	for _, d := range ix.Docs {
		if _, err := os.Stat(d.Path); os.IsNotExist(err) {
			return true
		}
	}
	return false
}

// inGitWorktree reports whether dir is inside a linked git worktree (as opposed
// to a main working tree): a worktree's --git-dir and its shared --git-common-dir
// are different directories, a main repo's are the same one. The two are compared
// by identity (os.SameFile), not by path string — git returns one absolute
// (symlink-resolved) and one relative path, so a lexical string compare would
// wrongly flag a main repo reached through a symlinked path as a worktree and
// silently disable auto-indexing. A non-git directory is not a worktree.
func inGitWorktree(dir string) bool {
	gd, e1 := gitRevParsePath(dir, "--git-dir")
	cd, e2 := gitRevParsePath(dir, "--git-common-dir")
	if e1 != nil || e2 != nil {
		return false
	}
	gi, e1 := os.Stat(gd)
	ci, e2 := os.Stat(cd)
	if e1 != nil || e2 != nil {
		return false
	}
	return !os.SameFile(gi, ci)
}

func gitRevParsePath(dir, flag string) (string, error) {
	out, err := exec.Command("git", "-C", dir, "rev-parse", flag).Output()
	if err != nil {
		return "", err
	}
	p := strings.TrimSpace(string(out))
	if !filepath.IsAbs(p) {
		p = filepath.Join(dir, p) // a relative git path is relative to dir
	}
	return p, nil
}

func init() {
	// 15 minutes was correct for the English embedder it was chosen with. The
	// multilingual model that replaced it is 2.7x slower per chunk (1.841s vs
	// 0.673s, same machine, back to back), which took a cold build of this
	// project's corpus from ~17 to ~45 minutes — so the old default cut a cold
	// build off at roughly a third and, before the guard above, saved the wreckage
	// over a good index and reported success.
	//
	// 90 minutes is that measurement doubled. The exact number matters much less
	// now than it did: expiry is a refusal, not a corruption, so being wrong here
	// costs a re-run rather than an index nobody can tell is broken. On a small
	// corpus the build finishes in seconds and the deadline never fires at all.
	//
	// It is still a measurement with an expiry date. It tracks the model AND the
	// corpus size, both of which move — re-derive it if either changes, and do not
	// read 90 as a property of the code.
	retrieveIndexCmd.Flags().Duration("timeout", 90*time.Minute, "overall deadline for the index build")
	retrieveIndexCmd.Flags().Bool("lexical", false, "build a BM25-only index without the semantic embedder")
	retrieveWarmCmd.Flags().Duration("timeout", 5*time.Minute, "overall deadline for the model download + warm")
	retrieveCmd.Flags().String("query", "", "search with this query instead of reading a hook payload on stdin")
	retrieveCmd.AddCommand(retrieveIndexCmd, retrieveWarmCmd, retrieveStatsCmd, retrieveTurnEndCmd)
}

// retrieveTurnEndCmd is a Stop hook that writes one line per completed turn and
// prints nothing. It exists to give consultation an honest DENOMINATOR.
//
// Without it the only available rate is "reads per offered page", which is
// meaningless: the hook offers 5 pages on every prompt including the ~95% that
// carry no question for the corpus, so a low rate says nothing about whether
// consultation is working. Turns are the denominator a human would actually use
// — "in how many turns did the agent check the record?" — and nothing else in
// the system records that a turn ended.
//
// Log-only by decision. A Stop hook's stdout does not reach the model anyway
// (this repo has a page on which hook events inject context), and the blocking
// variant — refusing to end a turn that consulted nothing — stays on the shelf
// until a measured miss rate justifies paying for a re-turn. Observing first is
// also the cheaper half: it replaces a forensic pass over a 30 MB transcript,
// which is how the 3.3% consultation rate had to be recovered.
var retrieveTurnEndCmd = &cobra.Command{
	Use:          "turn-end",
	Short:        "Record that a turn completed (Stop hook; writes to the fire log, prints nothing)",
	Hidden:       true,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		defer func() { _ = recover() }()
		if isATLInternalSession() {
			return nil
		}
		if root, err := projectKey(); err == nil {
			logRetrieveFire(root, "turn", nil)
		}
		return nil // never fail a turn over bookkeeping
	},
}

var retrieveStatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "What the retrieval hook has actually done in this project",
	Long: "Tally the per-fire log the hook writes beside the index: how many prompts it\n" +
		"ranked, how many produced nothing, how many were suppressed before ranking, and\n" +
		"which pages it has offered.\n\n" +
		"This is the subsystem's denominator. A high silent count is not a fault — it is\n" +
		"the retriever declining to answer a question it has no signal for.",
	Args:         cobra.NoArgs,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		root, err := projectKey()
		if err != nil {
			return err
		}
		idx, err := indexPathFor(root)
		if err != nil {
			return err
		}
		st, err := readFireStats(filepath.Join(filepath.Dir(idx), "fires.log"))
		if err != nil {
			return err
		}
		fmt.Fprint(cmd.OutOrStdout(), renderFireStats(st))
		return nil
	},
}

// renderFireStats formats the tally. Separated from the command so the numbers a
// human will act on are asserted rather than eyeballed once — this output is the
// evidence for whether injected context works as a delivery channel at all, so a
// wrong denominator here would mis-decide that.
func renderFireStats(st retrieveFireStats) string {
	if st.Total == 0 {
		return "atl retrieve: no fires recorded yet for this project\n"
	}
	pct := func(n int) string { return fmt.Sprintf("%5.1f%%", 100*float64(n)/float64(st.Total)) }
	var b strings.Builder
	fmt.Fprintf(&b, "fires        %5d\n", st.Total)
	fmt.Fprintf(&b, "  ranked     %5d  %s\n", st.Fired+st.Silent, pct(st.Fired+st.Silent))
	fmt.Fprintf(&b, "    offered  %5d  %s   (%d page refs)\n", st.Fired, pct(st.Fired), st.Offered)
	fmt.Fprintf(&b, "    silent   %5d  %s\n", st.Silent, pct(st.Silent))
	fmt.Fprintf(&b, "  suppressed %5d  %s   (machine %d, short %d)\n",
		st.Machine+st.Short, pct(st.Machine+st.Short), st.Machine, st.Short)
	if st.Translated > 0 {
		fmt.Fprintf(&b, "  translated %5d  %s   (of the above — a query with no lexical hit)\n",
			st.Translated, pct(st.Translated))
		if st.TranslatedCached > 0 {
			fmt.Fprintf(&b, "    from cache %5d       (no session spawned — the usage budget this did not spend)\n",
				st.TranslatedCached)
		}
	}
	// The spend, stated rather than left as arithmetic for the reader.
	//
	// Every one of these is a `claude` session against the SAME weekly budget the
	// user's own sessions draw on. That sentence is the entire reason this line
	// exists: 214 of them accumulated across projects, unnoticed, during the week the
	// maintainer exhausted one account — and nothing anywhere said so. A number a
	// reader has to derive by subtracting two others is a number nobody derives.
	if spawned := st.Translated - st.TranslatedCached + st.TranslateFailed; spawned > 0 {
		fmt.Fprintf(&b, "  sessions spawned %5d       (translation spends the same weekly budget your own sessions do)\n",
			spawned)
	}
	// Shown because it is the one state a user cannot otherwise see: translation is
	// fail-open, so a failure produces no error, no missing output, and no change a
	// reader would notice — only a silently worse search. The session-start notice
	// needs three consecutive failures before it speaks; this line shows the state
	// from the first one.
	//
	// Broken out by REASON, and the parenthetical no longer guesses. It used to read
	// "credential expired?" for every failure; the condition actually occurring was
	// an exhausted weekly quota, and that guess cost a diagnosis — the credential was
	// probed and found healthy before anyone read the child's own error text.
	if st.TranslateFailed > 0 {
		fmt.Fprintf(&b, "  translate-failed %5d\n", st.TranslateFailed)
		for _, r := range translateFailureOrder {
			n := st.TranslateFailedBy[r.key]
			if n == 0 {
				continue
			}
			fmt.Fprintf(&b, "    %-14s %5d       %s\n", r.key, n, r.remedy)
		}
	}
	// The agent-initiated half, reported against TURNS rather than against fires.
	// "Reads per offered page" is the wrong rate — the hook offers pages on every
	// prompt including the ones carrying no question — so the number that answers
	// "is the agent checking the record?" needs turns underneath it.
	if st.Turns > 0 || st.Consulted > 0 || st.ConsultedEmpty > 0 {
		asked := st.Consulted + st.ConsultedEmpty
		fmt.Fprintf(&b, "\nturns        %5d\n", st.Turns)
		if st.Turns > 0 {
			fmt.Fprintf(&b, "  consulted  %5d  %5.1f%%   (agent wrote its own query)\n",
				asked, 100*float64(asked)/float64(st.Turns))
		} else {
			fmt.Fprintf(&b, "  consulted  %5d          (no turn recorded yet — is the Stop hook installed?)\n", asked)
		}
		if st.ConsultedEmpty > 0 {
			fmt.Fprintf(&b, "    no match %5d          (asked, corpus had nothing — a gap, not a miss)\n", st.ConsultedEmpty)
		}
	}
	if len(st.Pages) == 0 {
		return b.String()
	}
	type kv struct {
		p string
		n int
	}
	top := make([]kv, 0, len(st.Pages))
	for p, n := range st.Pages {
		top = append(top, kv{p, n})
	}
	sort.Slice(top, func(a, b int) bool {
		if top[a].n != top[b].n {
			return top[a].n > top[b].n
		}
		return top[a].p < top[b].p
	})
	fmt.Fprintf(&b, "\ndistinct pages offered: %d\n", len(top))
	for i, t := range top {
		if i == 10 {
			fmt.Fprintf(&b, "  … %d more\n", len(top)-10)
			break
		}
		fmt.Fprintf(&b, "  %4d  %s  %s\n", t.n, pct(t.n), t.p)
	}
	return b.String()
}
