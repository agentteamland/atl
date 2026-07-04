package commands

import (
	"errors"

	"github.com/spf13/cobra"
)

// workCmd groups the delivery-team orchestration commands. It is HIDDEN: `atl
// work dispatch` is an internal engine invoked by the delivery-team's ceremony
// skills (/sprint-start), not typed by users, and the delivery-team is not yet
// shipped — so it stays out of `atl --help` and out of the docs-coverage gate
// until it ships (commandDocs skips Hidden commands — see docs.go).
var workCmd = &cobra.Command{
	Use:    "work",
	Short:  "Delivery-team work orchestration (internal)",
	Long: "Orchestration engine for the delivery-team: schedule a sprint's work-units\n" +
		"across isolated `claude -p` workers in git worktrees. Invoked by the ceremony\n" +
		"skills, not run by hand.",
	Hidden: true,
}

var workDispatchCmd = &cobra.Command{
	Use:   "dispatch",
	Short: "Run a sprint's work-unit DAG across a pool of isolated workers",
	Long: "Read the materialized plan (.delivery/plan.json) a /sprint-start ceremony\n" +
		"wrote, then schedule its work-units over a fixed-cap pool of `claude -p`\n" +
		"workers — each in its own git worktree off `dev`, observed through the\n" +
		"status.json each worker writes. Deterministic supervisor: no LLM context,\n" +
		"no Azure calls (the workers own the Azure MCP surface).",
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return errors.New("not yet implemented")
	},
}

func init() {
	workCmd.AddCommand(workDispatchCmd)
}
