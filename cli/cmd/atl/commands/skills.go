package commands

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/agentteamland/atl/cli/internal/scope"
	"github.com/agentteamland/atl/cli/internal/skillcheck"
	"github.com/agentteamland/atl/cli/internal/sweepstate"
	"github.com/spf13/cobra"
)

var skillsCmd = &cobra.Command{
	Use:   "skills",
	Short: "Content-quality checks for the platform's skills, agents, and teams",
	Long: "Deterministic, LLM-free content-quality checks for the repo's own skills,\n" +
		"agents, and team manifests — the sibling of `atl docs check`. docs check\n" +
		"validates the docs site; skills check validates the assets themselves. The\n" +
		"judgment half (obedience, redundancy) lives in the /skill-stocktake skill.",
}

var skillsCheckCmd = &cobra.Command{
	Use:   "check",
	Short: "Validate skill/agent frontmatter, team.json consistency, agent-KB children, and skill shell bodies",
	Long: "Check the repo's assets for content-quality drift: every skill/agent carries a\n" +
		"name + description frontmatter, each team.json matches its on-disk agents and\n" +
		"skills (both directions), every agent-KB child declares its summary, and no\n" +
		"skill's fenced shell body uses a construct that aborts under zsh. Exits\n" +
		"non-zero on any failure (warnings never fail). Outside the monorepo it does\n" +
		"nothing and exits 0 (the pre-flight skip).",
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		record, _ := cmd.Flags().GetBool("record-stocktake")

		root, err := findCoreRoot()
		if err != nil {
			fmt.Println("atl skills: no core/ here — nothing to check (set ATL_REPO_ROOT if the monorepo is not an ancestor of this directory)")
			return nil
		}
		in := skillcheck.Input{
			CoreDir:  filepath.Join(root, "core"),
			TeamsDir: filepath.Join(root, "teams"),
		}
		findings := skillcheck.RunAll(in)

		fails, warns := 0, 0
		for _, f := range findings {
			marker := "FAIL"
			if f.Severity == skillcheck.Warn {
				marker, warns = "warn", warns+1
			} else {
				fails++
			}
			loc := f.Path
			if loc == "" {
				loc = "-"
			}
			fmt.Printf("  [%s] %s · %s — %s\n", marker, f.Check, loc, f.Detail)
		}

		if record && fails == 0 {
			if sha := gitHEAD(root); sha != "" {
				_ = sweepstate.Skills.Record(sha, time.Now())
			}
		}

		switch {
		case fails > 0:
			return fmt.Errorf("%d skill/asset quality item(s), %d warning(s) — fix before shipping", fails, warns)
		case warns > 0:
			fmt.Printf("atl skills: no failures (%d warning(s))\n", warns)
		default:
			fmt.Println("atl skills: clean")
		}
		return nil
	},
}

// findCoreRoot locates the monorepo root — the dir holding core/skills — from
// the cwd, upward and then a bounded search downward (see findRepoRoot).
// Returns an error outside the monorepo, which is the pre-flight skip.
func findCoreRoot() (string, error) {
	return findRepoRoot(filepath.Join("core", "skills"))
}

// skillsSessionSignal surfaces asset content-quality signals at session start, but
// only inside the monorepo (findCoreRoot succeeds) — silent in end-user projects,
// so those sessions pay nothing. Two signals: a deterministic drift count (the
// fast Layer) and, when a full sweep is due, a /skill-stocktake signal (the LLM
// backstop). Best-effort; a hook must never block.
func skillsSessionSignal() {
	root, err := findCoreRoot()
	if err != nil {
		return // not the monorepo — the pre-flight skip
	}
	in := skillcheck.Input{
		CoreDir:  filepath.Join(root, "core"),
		TeamsDir: filepath.Join(root, "teams"),
	}
	fails := 0
	for _, f := range skillcheck.RunAll(in) {
		if f.Severity == skillcheck.Fail {
			fails++
		}
	}
	if fails > 0 {
		fmt.Printf("atl skills: %d asset quality item(s) — run `atl skills check`\n", fails)
	}
	if sweepstate.Skills.Due(root) {
		fmt.Println(sweepNotice("atl skills: a stocktake is due", "/skill-stocktake", "sweep skills for obedience and redundancy"))
	}
}

// installedChildrenSignal warns when an agent-KB child in an INSTALLED layer
// (~/.claude or <project>/.claude) carries no knowledge-base-summary.
//
// Unlike the docs/skills/rules signals above this is deliberately NOT
// monorepo-gated: /drain writes agent-KB children in any project, and that
// surface is precisely the one `atl skills check` cannot see — it walks the
// shipped teams/ copies, and CI has no installed layer to gate on. Warn-only:
// the frontmatter is missing, which is silent, not dangerous. Best-effort; a
// hook must never block.
func installedChildrenSignal(projectRoot string) {
	n := 0
	seen := map[string]bool{}
	for _, sc := range []scope.Scope{scope.Global, scope.Project} {
		if sc == scope.Project && projectRoot == "" {
			continue
		}
		dir, err := scope.ClaudeDir(sc, projectRoot)
		if err != nil {
			continue
		}
		// The project root is the session's cwd, so a session started in $HOME
		// resolves BOTH layers to ~/.claude — count that dir once, or every
		// finding there is reported twice.
		if seen[dir] {
			continue
		}
		seen[dir] = true
		n += len(skillcheck.InstalledChildren(dir))
	}
	if n > 0 {
		fmt.Printf("atl: %d agent-KB child(ren) under .claude/agents/*/children/ carry no `knowledge-base-summary` — an agent's `## Knowledge Base` section is derived from that frontmatter, so there is nothing to rebuild their entries from; add it\n", n)
	}
}

func init() {
	skillsCheckCmd.Flags().Bool("record-stocktake", false, "record HEAD as the last-stocktaken commit when free of failures")
	skillsCmd.AddCommand(skillsCheckCmd)
}
