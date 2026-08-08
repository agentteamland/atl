package commands

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/agentteamland/atl/cli/internal/digest"
	"github.com/spf13/cobra"
)

var digestCmd = &cobra.Command{
	Use:   "digest",
	Short: "Sweep findings that are waiting for a human decision",
	Long: "The half of a sweep's output that cannot be carded. `sweep-dispatch` splits a\n" +
		"sweep's findings by whether they need a decision: actionable-and-already-decided\n" +
		"goes to the board, which already carries work to closure; needing-judgement comes\n" +
		"here, because a finding that needs a decision needs it before it needs a ticket.\n\n" +
		"Run bare, it prints the unread findings and marks them read.",
}

var digestShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Print the findings waiting for a decision, and mark them read",
	Long: "Print the unread findings and mark them read. `--all` prints every finding,\n" +
		"read ones included, and marks nothing.\n\n" +
		"Reading is not deciding, so a finding stays in the digest after it is shown —\n" +
		"use `atl digest drop <id>` once it has actually been settled.",
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		all, _ := cmd.Flags().GetBool("all")
		root, err := projectKey()
		if err != nil {
			return err
		}
		findings, err := digest.Load(root)
		if err != nil {
			return err
		}
		shown := 0
		for _, f := range findings {
			if !all && !f.Unread() {
				continue
			}
			state := ""
			if !all && f.Unread() {
				state = ""
			} else if f.Unread() {
				state = "  [unread]"
			}
			fmt.Printf("\n%s · %s%s\n  %s\n", f.ID, f.Sweep, state, f.Title)
			for _, line := range strings.Split(strings.TrimRight(f.Body, "\n"), "\n") {
				fmt.Printf("    %s\n", line)
			}
			shown++
		}
		if shown == 0 {
			fmt.Println("atl digest: nothing waiting")
			return nil
		}
		if !all {
			if n, merr := digest.MarkRead(root); merr == nil && n > 0 {
				fmt.Printf("\natl digest: %d finding(s) marked read — `atl digest drop <id>` once decided\n", n)
			}
		}
		return nil
	},
}

var digestAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Record a finding that needs a decision (the body is read from stdin)",
	Long: "Record a finding for the user to decide on. Written by a sweep; the body is\n" +
		"read from stdin so it can carry evidence without shell quoting.\n\n" +
		"Idempotent by (sweep, title): re-reporting a finding refreshes its body and\n" +
		"leaves its read state alone, so a sweep that runs daily converges instead of\n" +
		"re-interrupting the reader about the same thing.",
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		sweep, _ := cmd.Flags().GetString("sweep")
		title, _ := cmd.Flags().GetString("title")
		if strings.TrimSpace(sweep) == "" || strings.TrimSpace(title) == "" {
			return fmt.Errorf("atl digest add: --sweep and --title are both required")
		}
		body, err := io.ReadAll(bufio.NewReader(cmd.InOrStdin()))
		if err != nil {
			return err
		}
		root, err := projectKey()
		if err != nil {
			return err
		}
		added, err := digest.Add(root, digest.Finding{
			Sweep: strings.TrimSpace(sweep),
			Title: strings.TrimSpace(title),
			Body:  strings.TrimRight(string(body), "\n"),
		})
		if err != nil {
			return err
		}
		if added {
			fmt.Printf("atl digest: recorded %s\n", digest.NewID(sweep, title))
		} else {
			fmt.Printf("atl digest: %s already recorded — refreshed, read state kept\n", digest.NewID(sweep, title))
		}
		return nil
	},
}

var digestDropCmd = &cobra.Command{
	Use:   "drop <id>",
	Short: "Remove a finding that has been decided on",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		root, err := projectKey()
		if err != nil {
			return err
		}
		dropped, err := digest.Drop(root, args[0])
		if err != nil {
			return err
		}
		if !dropped {
			return fmt.Errorf("atl digest: no finding with id %s", args[0])
		}
		fmt.Printf("atl digest: dropped %s\n", args[0])
		return nil
	},
}

func init() {
	digestShowCmd.Flags().Bool("all", false, "print every finding, read ones included, and mark nothing")
	digestAddCmd.Flags().String("sweep", "", "the sweep reporting this finding (observe, skill-stocktake, ...)")
	digestAddCmd.Flags().String("title", "", "one line naming the finding — half of its stable key")

	digestCmd.AddCommand(digestShowCmd, digestAddCmd, digestDropCmd)
	// Bare `atl digest` is the reader's entry point; the sub-commands are for the
	// sweep that writes and the reader who settles.
	digestCmd.RunE = digestShowCmd.RunE
	digestCmd.Args = cobra.NoArgs
	digestCmd.Flags().Bool("all", false, "print every finding, read ones included, and mark nothing")
	rootCmd.AddCommand(digestCmd)
}
