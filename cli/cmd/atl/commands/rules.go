package commands

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"

	"github.com/agentteamland/atl/cli/internal/rulesscan"
	"github.com/agentteamland/atl/cli/internal/rulesstate"
	"github.com/spf13/cobra"
)

var rulesCmd = &cobra.Command{
	Use:   "rules",
	Short: "Corpus-driven rule discovery — the deterministic collect for /rules-distill",
	Long: "The deterministic collect half of rules-distill: gather the normative /\n" +
		"imperative statements across the skill + agent corpus so the /rules-distill\n" +
		"skill can judge which recurring principles deserve to become a core rule. The\n" +
		"judgment (which candidate is a real principle) is the skill's — the CLI/Skill\n" +
		"boundary.",
}

var rulesScanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Collect normative statements across the skill corpus (for /rules-distill)",
	Long: "Walk the skill + agent markdown in core/ and teams/ and print every line\n" +
		"carrying a normative/imperative trigger (always, never, must, prefer, …), with\n" +
		"its source location — the grounded candidate list /rules-distill clusters. It\n" +
		"over-collects on purpose; the LLM decides which candidates are real principles.\n" +
		"rules/ subtrees are skipped (they're the distill target, not a source). Outside\n" +
		"the monorepo it does nothing and exits 0.",
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		record, _ := cmd.Flags().GetBool("record")
		jsonOut, _ := cmd.Flags().GetBool("json")

		root, err := findCoreRoot()
		if err != nil {
			fmt.Println("atl rules: no core/ here — nothing to scan")
			return nil
		}
		stmts, err := rulesscan.Collect(filepath.Join(root, "core"), filepath.Join(root, "teams"))
		if err != nil {
			return err
		}

		if record {
			if sha := gitHEAD(root); sha != "" {
				_ = rulesstate.Record(sha, time.Now())
			}
			fmt.Printf("atl rules: recorded HEAD as the last distill (%d candidate statement(s))\n", len(stmts))
			return nil
		}
		if jsonOut {
			b, err := json.MarshalIndent(stmts, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(b))
			return nil
		}
		if len(stmts) == 0 {
			fmt.Println("atl rules: no normative statements found")
			return nil
		}
		for _, s := range stmts {
			fmt.Printf("%s:%d  %s\n", s.File, s.Line, s.Text)
		}
		return nil
	},
}

// rulesSessionSignal surfaces a "distill due" signal at session start, but only
// inside the monorepo — silent for end users. Non-destructive (propose-only), so
// it auto-signals per the Lane 3 automation decision. Best-effort; never blocks.
func rulesSessionSignal() {
	root, err := findCoreRoot()
	if err != nil {
		return
	}
	if rulesDistillDue(root) {
		fmt.Println("atl rules: a distill is due — run /rules-distill to mine recurring principles into core rules")
	}
}

// rulesDistillDue reports whether a /rules-distill sweep is due: corpus-affecting
// commits (core/ or teams/) have landed since the last recorded distill, gated by
// a ~1-day runaway-guard. Mirrors stocktakeDue / docsAuditDue.
func rulesDistillDue(repoRoot string) bool {
	st, err := rulesstate.Load()
	if err != nil {
		return false
	}
	if st.LastDistillAt != "" {
		if t, perr := time.Parse(time.RFC3339, st.LastDistillAt); perr == nil && time.Since(t) < 24*time.Hour {
			return false
		}
	}
	return assetAffectingCommitsSince(repoRoot, st.LastDistillSHA)
}

func init() {
	rulesScanCmd.Flags().Bool("record", false, "record HEAD as the last-distilled commit (after a /rules-distill sweep)")
	rulesScanCmd.Flags().Bool("json", false, "emit statements as JSON")
	rulesCmd.AddCommand(rulesScanCmd)
}
