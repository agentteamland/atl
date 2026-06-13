package commands

import (
	"fmt"
	"os"

	"github.com/agentteamland/atl/cli/internal/queue"
	"github.com/spf13/cobra"
)

var learningsCmd = &cobra.Command{
	Use:   "learnings",
	Short: "Inspect the learning queue",
	Long: "Inspect the durable learning queue — the substrate the self-driving loop\n" +
		"runs on. Markers captured in conversation are transferred into the queue\n" +
		"exactly once, processed, then deleted (so they can never be re-reported).",
}

var learningsStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show pending queue items per channel / project",
	Long: "Read pending counts straight from the queue — correct by construction,\n" +
		"never inferred from re-scanning. This is what the SessionStart count uses.",
	RunE: func(cmd *cobra.Command, args []string) error {
		st, project, err := openQueue()
		if err != nil {
			return err
		}
		defer st.Close()

		counts, err := st.Counts(project)
		if err != nil {
			return err
		}
		if len(counts) == 0 {
			fmt.Println("learning queue: empty (nothing pending)")
			return nil
		}
		fmt.Println("learning queue — pending by channel:")
		for _, ch := range []queue.Channel{queue.ChannelLearning, queue.ChannelProfileFact} {
			if n, ok := counts[ch]; ok {
				fmt.Printf("  %-14s %d\n", string(ch), n)
				delete(counts, ch)
			}
		}
		for ch, n := range counts { // any future channels
			fmt.Printf("  %-14s %d\n", string(ch), n)
		}
		return nil
	},
}

// learningsEnqueueCmd is a hidden helper: hooks (and tests) use it to transfer
// a captured marker into the queue exactly once. The dedup lives in the store,
// so calling it twice with the same marker is a safe no-op.
var learningsEnqueueCmd = &cobra.Command{
	Use:    "_enqueue <channel> <payload>",
	Short:  "(internal) transfer a marker into the queue",
	Hidden: true,
	Args:   cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		st, project, err := openQueue()
		if err != nil {
			return err
		}
		defer st.Close()

		ch := queue.Channel(args[0])
		payload := args[1]
		added, err := st.Enqueue(project, queue.Item{
			ID:      queue.NewID(ch, payload),
			Channel: ch,
			Payload: payload,
		})
		if err != nil {
			return err
		}
		if added {
			fmt.Println("enqueued")
		} else {
			fmt.Println("already queued (dedup)")
		}
		return nil
	},
}

// openQueue opens the default queue and resolves the current project key (cwd).
func openQueue() (*queue.Store, string, error) {
	dbPath, err := queue.DefaultPath()
	if err != nil {
		return nil, "", err
	}
	st, err := queue.Open(dbPath)
	if err != nil {
		return nil, "", err
	}
	project, err := os.Getwd()
	if err != nil {
		_ = st.Close()
		return nil, "", err
	}
	return st, project, nil
}

func init() {
	learningsCmd.AddCommand(learningsStatusCmd, learningsEnqueueCmd)
}
