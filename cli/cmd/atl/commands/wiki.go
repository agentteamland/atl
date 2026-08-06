package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/agentteamland/atl/cli/internal/wikicheck"
	"github.com/spf13/cobra"
)

var wikiCmd = &cobra.Command{
	Use:   "wiki",
	Short: "Integrity checks for this project's .atl/wiki current-truth layer",
	Long: "Deterministic, LLM-free integrity checks for `.atl/wiki` — the sibling of\n" +
		"`atl docs check` and `atl skills check`, over the knowledge layer rather than\n" +
		"the docs site or the assets.\n\n" +
		"It deliberately does NOT check whether a page is still TRUE. The docs site can\n" +
		"be checked for drift because it has a ground truth to compare against — the\n" +
		"code. A wiki page's claim has no single referent, and the obvious candidate\n" +
		"check (does a cited repo path still exist?) measured 76-90% false positive,\n" +
		"because documenting that something is dead is byte-identical to citing\n" +
		"something dead. That judgment half lives in the /observe sweep.",
}

var wikiCheckCmd = &cobra.Command{
	Use:   "check",
	Short: "Validate that wiki pages are reachable from CLAUDE.md and that their links resolve",
	Long: "Check this project's knowledge layer for the two things that ARE decidable:\n" +
		"every CLAUDE.md link into `.atl/wiki` resolves on disk, every wiki page is\n" +
		"linked from CLAUDE.md at all, and every relative link inside a page resolves.\n\n" +
		"Reachability is not tidiness: the CLAUDE.md index is what a generated\n" +
		"retrieval query draws its vocabulary from, measured at 100% recall@5 with the\n" +
		"index loaded against 58% without it. A page nothing links to is outside that\n" +
		"vocabulary, so it is not merely harder to find — it is unreachable by the\n" +
		"mechanism built to find it.\n\n" +
		"Exits non-zero on any finding. In a project with no `.atl/wiki` it does\n" +
		"nothing and exits 0.",
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		root, err := os.Getwd()
		if err != nil {
			return err
		}
		if fi, serr := os.Stat(filepath.Join(root, ".atl", "wiki")); serr != nil || !fi.IsDir() {
			fmt.Println("atl wiki: no .atl/wiki here — nothing to check")
			return nil
		}

		findings := wikicheck.RunAll(wikicheck.Input{ProjectRoot: root})
		for _, f := range findings {
			loc := f.Path
			if loc == "" {
				loc = "-"
			}
			fmt.Printf("  [FAIL] %s · %s — %s\n", f.Check, loc, f.Detail)
		}
		if len(findings) > 0 {
			return fmt.Errorf("atl wiki: %d finding(s)", len(findings))
		}
		fmt.Println("atl wiki: clean")
		return nil
	},
}

func init() {
	wikiCmd.AddCommand(wikiCheckCmd)
	rootCmd.AddCommand(wikiCmd)
}
