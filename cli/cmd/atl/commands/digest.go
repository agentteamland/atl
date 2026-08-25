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
			// The footer belongs on THIS path too, and it was originally placed only
			// after it. "Nothing waiting here, and 56 findings in four other stores"
			// is the single most useful thing this command can say — and an empty
			// digest is exactly when a reader most needs telling that the rest exist.
			// Returning early made the message unreachable in its best case.
			printOtherStores(root)
			return nil
		}
		if !all {
			if n, merr := digest.MarkRead(root); merr == nil && n > 0 {
				fmt.Printf("\natl digest: %d finding(s) marked read — `atl digest drop <id>` once decided\n", n)
			}
		}
		printOtherStores(root)
		return nil
	},
}

// printOtherStores says that the other digests exist, and nothing more.
//
// The split itself is correct — one store per project, so whichever project was
// opened first cannot answer for the rest. What was wrong is that it was SILENT:
// a hub that clones repos beneath it gives each its own store, a sweep run inside
// one writes there, and the hub's digest goes on answering normally with no
// absence to notice. Measured on one machine: six stores, 73 findings, of which a
// hub session saw 17 — and nine findings about the platform's own skills sat in
// <hub>/repos/atl, reachable and never reached.
//
// It prints ONLY when another store exists. A footer on every run is the
// constant-channel shape this package's own header rejects, and it would be
// wallpaper on the overwhelmingly common single-project machine.
func printOtherStores(root string) {
	stores, err := digest.Stores()
	if err != nil {
		return
	}
	// Deliberately NOT `len(stores) < 2`. That was the first guard here and it was
	// wrong in the case this exists for: a project with no digest of its own has
	// exactly one store on disk — somebody else's — and is precisely the reader who
	// needs telling. The real question is how many stores are not mine, which the
	// count below answers.
	mine, err := digest.Path(root)
	if err != nil {
		return
	}
	others, total, unread := 0, 0, 0
	for _, s := range stores {
		if s.Path == mine {
			continue
		}
		others++
		total += s.Total
		unread += s.Unread
	}
	if others == 0 {
		return
	}
	fmt.Printf("\natl digest: %d other project digest(s) on this machine hold %d finding(s), %d unread.\n",
		others, total, unread)
	fmt.Printf("            They are not shown here — a digest answers for its own project.\n")
	fmt.Printf("            `atl digest projects` lists them.\n")
}

var digestProjectsCmd = &cobra.Command{
	Use:   "projects",
	Short: "List every digest on this machine and the project each belongs to",
	Long: "List every digest store under ~/.atl/digest, with its project and its counts.\n\n" +
		"A digest is per project on purpose, so one project cannot answer for the rest.\n" +
		"The cost of that is a split nothing reported: a hub that clones repos beneath\n" +
		"itself gives each of them its own store, and the hub goes on answering normally\n" +
		"while findings accumulate somewhere nobody opens a session.\n\n" +
		"A store written before the project root was recorded shows as `not recorded`.\n" +
		"That is not guessed at — a reverse lookup would manufacture a path carrying more\n" +
		"confidence than the file has.",
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		stores, err := digest.Stores()
		if err != nil {
			return err
		}
		if len(stores) == 0 {
			fmt.Println("atl digest: no digests on this machine")
			return nil
		}
		mine := ""
		if root, rerr := projectKey(); rerr == nil {
			if p, perr := digest.Path(root); perr == nil {
				mine = p
			}
		}
		total, unread, unnamed := 0, 0, 0
		for _, s := range stores {
			here := "  "
			if s.Path == mine {
				here = "* "
			}
			root := s.Root
			if root == "" {
				root = "(project not recorded — written before this was tracked)"
				unnamed++
			}
			fmt.Printf("%s%s  %3d finding(s), %d unread  %s\n", here, s.Key, s.Total, s.Unread, root)
			total += s.Total
			unread += s.Unread
		}
		fmt.Printf("\n%d store(s), %d finding(s), %d unread. `*` is this project.\n", len(stores), total, unread)
		if unnamed > 0 {
			fmt.Printf("%d store(s) do not record their project. Anything written to them from now on will.\n", unnamed)
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

	digestCmd.AddCommand(digestShowCmd, digestAddCmd, digestDropCmd, digestProjectsCmd)
	// Bare `atl digest` is the reader's entry point; the sub-commands are for the
	// sweep that writes and the reader who settles.
	digestCmd.RunE = digestShowCmd.RunE
	digestCmd.Args = cobra.NoArgs
	digestCmd.Flags().Bool("all", false, "print every finding, read ones included, and mark nothing")
	rootCmd.AddCommand(digestCmd)
}
