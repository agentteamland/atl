package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/agentteamland/atl/cli/internal/skillcheck"
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

// skillsSessionSignal surfaces asset content-quality drift at session start, but
// only inside the monorepo (findCoreRoot succeeds) — silent in end-user projects,
// so those sessions pay nothing. Best-effort; a hook must never block.
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
}

func init() {
	skillsCmd.AddCommand(skillsCheckCmd)
}
