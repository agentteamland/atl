package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/agentteamland/atl/cli/internal/skillcheck"
	"github.com/agentteamland/atl/cli/internal/skillsstate"
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
	Short: "Validate skill/agent frontmatter, team.json consistency, and agent-KB children",
	Long: "Check the repo's assets for content-quality drift: every skill/agent carries a\n" +
		"name + description frontmatter, each team.json matches its on-disk agents and\n" +
		"skills (both directions), and every agent-KB child declares its summary. Exits\n" +
		"non-zero on any failure (warnings never fail). Outside the monorepo it does\n" +
		"nothing and exits 0 (the pre-flight skip).",
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		record, _ := cmd.Flags().GetBool("record-stocktake")

		root, err := findCoreRoot()
		if err != nil {
			fmt.Println("atl skills: no core/ here — nothing to check")
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
				_ = skillsstate.Record(sha, time.Now())
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

// findCoreRoot walks up from the cwd to the monorepo root — the dir holding
// core/skills. Returns an error outside the monorepo (the pre-flight skip).
func findCoreRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if st, e := os.Stat(filepath.Join(dir, "core", "skills")); e == nil && st.IsDir() {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no core/skills found from %s upward", dir)
		}
		dir = parent
	}
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
	if stocktakeDue(root) {
		fmt.Println("atl skills: a stocktake is due — run /skill-stocktake to sweep skills for obedience + redundancy")
	}
}

// stocktakeDue reports whether a /skill-stocktake full sweep is due:
// asset-affecting commits (core/ or teams/) have landed since the last recorded
// stocktake, gated by a ~1-day runaway-guard. Mirrors docsAuditDue.
func stocktakeDue(repoRoot string) bool {
	st, err := skillsstate.Load()
	if err != nil {
		return false
	}
	if st.LastStocktakeAt != "" {
		if t, perr := time.Parse(time.RFC3339, st.LastStocktakeAt); perr == nil && time.Since(t) < 24*time.Hour {
			return false // runaway-guard: don't sweep again within ~1 day
		}
	}
	return assetAffectingCommitsSince(repoRoot, st.LastStocktakeSHA)
}

// assetAffectingCommitsSince reports whether any commit touching core/ or teams/
// has landed since sinceSHA (or, when sinceSHA is empty, whether the repo has any
// such commit at all — i.e. it was never stocktaken).
func assetAffectingCommitsSince(repoRoot, sinceSHA string) bool {
	run := func(sha string) ([]byte, error) {
		args := []string{"-C", repoRoot, "log", "--oneline", "-1"}
		if sha != "" {
			args = append(args, sha+"..HEAD")
		}
		args = append(args, "--", "core", "teams")
		return exec.Command("git", args...).Output()
	}
	out, err := run(sinceSHA)
	if err != nil {
		// A recorded SHA git no longer knows (rebase / squash / gc) makes the range
		// query fail — treat it as never-audited (retry with no range) so the stocktake
		// / distill signal doesn't go permanently silent, rather than reading the git
		// failure as "nothing changed".
		if sinceSHA == "" {
			return false
		}
		if out, err = run(""); err != nil {
			return false
		}
	}
	return len(strings.TrimSpace(string(out))) > 0
}

func init() {
	skillsCheckCmd.Flags().Bool("record-stocktake", false, "record HEAD as the last-stocktaken commit when free of failures")
	skillsCmd.AddCommand(skillsCheckCmd)
}
