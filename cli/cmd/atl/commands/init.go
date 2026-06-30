package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/agentteamland/atl/cli/internal/scaffold"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Scaffold a starter CLAUDE.md (project, global persona, or monorepo)",
	Long: "Drop a lean starter CLAUDE.md for the chosen tier — only if one doesn't\n" +
		"already exist, so your own CLAUDE.md is never overwritten.\n\n" +
		"  --project   (default) a project-root CLAUDE.md: stack / commands / conventions,\n" +
		"              leaving room for the /brainstorm + /drain managed marker blocks.\n" +
		"  --global    your personal ~/.claude/CLAUDE.md persona (ATL manages nothing in it).\n" +
		"  --monorepo  a lean ~30-line orientation file (pointers, not inlined content).\n\n" +
		"The three tiers, their token budgets, and the managed-vs-owned ownership model\n" +
		"are documented on the Claude Code conventions docs page.",
	RunE: func(cmd *cobra.Command, args []string) error {
		g, _ := cmd.Flags().GetBool("global")
		p, _ := cmd.Flags().GetBool("project")
		m, _ := cmd.Flags().GetBool("monorepo")
		if n := boolCount(g, p, m); n > 1 {
			return fmt.Errorf("--global, --project and --monorepo are mutually exclusive")
		}

		tier := scaffold.Project
		switch {
		case g:
			tier = scaffold.Global
		case m:
			tier = scaffold.Monorepo
		}

		root, err := os.Getwd()
		if err != nil {
			return err
		}
		name := ""
		if tier != scaffold.Global {
			name = filepath.Base(root)
		}

		path, created, err := scaffold.WriteIfAbsent(tier, root, name)
		if err != nil {
			return err
		}
		if created {
			fmt.Printf("atl: created %s (%s tier) — fill in the sections marked for you.\n", path, tier)
		} else {
			fmt.Printf("atl: %s already exists — left untouched.\n", path)
		}
		return nil
	},
}

func boolCount(bs ...bool) int {
	n := 0
	for _, b := range bs {
		if b {
			n++
		}
	}
	return n
}

func init() {
	initCmd.Flags().Bool("global", false, "scaffold the global persona ~/.claude/CLAUDE.md")
	initCmd.Flags().Bool("project", false, "scaffold a project-root CLAUDE.md (default)")
	initCmd.Flags().Bool("monorepo", false, "scaffold a lean monorepo orientation CLAUDE.md")
}
