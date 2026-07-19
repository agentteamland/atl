package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/agentteamland/atl/cli/internal/retrieve"
	"github.com/agentteamland/atl/cli/internal/scope"
	"github.com/agentteamland/atl/cli/internal/transcript"
	"github.com/spf13/cobra"
)

// retrieveTopK is how many knowledge pages the hook surfaces per prompt.
const retrieveTopK = 5

// retrieveInput is the UserPromptSubmit hook payload atl retrieve reads on stdin.
type retrieveInput struct {
	Prompt string `json:"prompt"`
	CWD    string `json:"cwd"`
}

var retrieveCmd = &cobra.Command{
	Use:    "retrieve",
	Short:  "Per-prompt knowledge retrieval (hybrid lexical + semantic)",
	Hidden: true, // internal — wired as a UserPromptSubmit hook, not typed by hand
	Long: "The read side of ATL's knowledge loop (the write side is learning-capture +\n" +
		"/drain). Run with no subcommand as a UserPromptSubmit hook: it reads the prompt\n" +
		"on stdin, ranks the project's knowledge pages (BM25 + a local semantic embedder,\n" +
		"RRF-fused), and prints the top matches as context. Fail-open — any error prints\n" +
		"nothing and never blocks the prompt.\n\n" +
		"  atl retrieve index   (re)build the index for the current project\n" +
		"  atl retrieve warm    download the embedding model and warm the pipeline",
	SilenceUsage: true,
	// The bare `atl retrieve` is the hook body.
	RunE: func(cmd *cobra.Command, args []string) error {
		runRetrieveHook(cmd)
		return nil // fail-open: never surface an error that would break the prompt
	},
}

// runRetrieveHook is the UserPromptSubmit hook. It is exhaustively fail-open:
// every failure path simply prints nothing, so a missing index, an absent model,
// or a parse error can never block or corrupt the prompt. The deferred recover is
// part of that contract — this hook fires on every prompt, and a panic deep in
// the embedder would exit non-zero and erase the prompt; nothing is printed until
// the single final write, so recovering here cleanly drops the retrieval instead.
func runRetrieveHook(cmd *cobra.Command) {
	defer func() { _ = recover() }()

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
	root := in.CWD
	if root == "" {
		root, _ = os.Getwd()
	}

	idxPath, err := indexPathFor(root)
	if err != nil {
		return
	}
	ix, err := retrieve.Load(idxPath)
	if err != nil || ix == nil {
		return // no index yet (built on drain) — nothing to inject
	}

	// The embedder is best-effort: if the model isn't downloaded or fails to load,
	// query with a nil embedder (BM25-only) rather than skipping retrieval entirely.
	ctx, cancel := context.WithTimeout(cmd.Context(), 5*time.Second)
	defer cancel()
	e := embedderIfModelPresent(ctx)
	if e != nil {
		defer e.Close()
	}

	results, err := ix.Query(ctx, prompt, e, retrieveTopK)
	if err != nil || len(results) == 0 {
		return
	}
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

		root, err := os.Getwd()
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

		t0 := time.Now()
		ix, err := retrieve.Build(ctx, docs, e)
		if err != nil {
			return err
		}
		idxPath, err := indexPathFor(root)
		if err != nil {
			return err
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

// corpusDirs returns the knowledge-corpus roots to index for a project: the
// project's own knowledge — the wiki (current truth) and journal (history).
// Missing dirs are skipped by the walker. The agent knowledge base and, for a
// delivery project, the in-repo docs/ store are deferred corpus expansions
// (they need frontmatter handling and, for global agents, cross-project
// relevance gating); v1 surfaces project knowledge, the core of what #140 exists
// for.
func corpusDirs(projectRoot string) ([]string, error) {
	atlDir, err := scope.LayerDir(scope.Project, projectRoot)
	if err != nil {
		return nil, err
	}
	return []string{
		filepath.Join(atlDir, "wiki"),
		filepath.Join(atlDir, "journal"),
	}, nil
}

// indexPathFor is the on-disk index for a project: a global cache keyed by the
// project path, so it is never committed and each project (and git worktree) has
// its own — ~/.atl/cache/retrieve/<project-slug>/index.gob.
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
	var b strings.Builder
	b.WriteString("[atl] Knowledge pages that may be relevant to this prompt (atl#140 retrieval).\n")
	b.WriteString("Consult any that match the topic before answering; open the path for the full page.\n")
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

func init() {
	retrieveIndexCmd.Flags().Duration("timeout", 15*time.Minute, "overall deadline for the index build")
	retrieveIndexCmd.Flags().Bool("lexical", false, "build a BM25-only index without the semantic embedder")
	retrieveWarmCmd.Flags().Duration("timeout", 5*time.Minute, "overall deadline for the model download + warm")
	retrieveCmd.AddCommand(retrieveIndexCmd, retrieveWarmCmd)
}
