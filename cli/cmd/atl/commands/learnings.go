package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/agentteamland/atl/cli/internal/queue"
	"github.com/spf13/cobra"
)

var learningsCmd = &cobra.Command{
	Use:   "learnings",
	Short: "Inspect and drain the learning queue",
	Long: "Inspect and drain the durable learning queue — the substrate the\n" +
		"self-driving loop runs on. Markers captured in conversation are transferred\n" +
		"into the queue exactly once; the /drain skill folds each into the knowledge\n" +
		"base (wiki / journal / agent KB) and acks it, so it's deleted and can never\n" +
		"be re-reported. peek + ack are the deterministic surface that skill uses.",
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

// learningsPeekCmd lists pending items for the /drain skill to process. The
// skill reads --json, integrates each item, then calls `ack <id>`.
var learningsPeekCmd = &cobra.Command{
	Use:   "peek",
	Short: "List pending queue items (for the /drain skill)",
	Long: "List pending items the /drain skill consumes: id, channel, payload.\n" +
		"--channel filters to one channel (e.g. learning); --json emits the full\n" +
		"machine-readable list the skill drives off of.",
	RunE: func(cmd *cobra.Command, args []string) error {
		st, project, err := openQueue()
		if err != nil {
			return err
		}
		defer st.Close()

		channel, _ := cmd.Flags().GetString("channel")
		items, err := st.Pending(project, queue.Channel(channel))
		if err != nil {
			return err
		}
		if jsonOut, _ := cmd.Flags().GetBool("json"); jsonOut {
			b, err := json.MarshalIndent(items, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(b))
			return nil
		}
		if len(items) == 0 {
			fmt.Println("no pending items")
			return nil
		}
		for _, it := range items {
			fmt.Printf("%-12s  %-12s  %s\n", it.ID[:12], it.Channel, firstLine(it.Payload))
		}
		return nil
	},
}

// learningsAckCmd deletes a processed item — processed-then-deleted, so it can
// never be re-reported. The /drain skill calls this after integrating an item.
var learningsAckCmd = &cobra.Command{
	Use:   "ack <id>",
	Short: "Mark an item processed (delete it from the queue)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		st, project, err := openQueue()
		if err != nil {
			return err
		}
		defer st.Close()
		if err := st.Delete(project, args[0]); err != nil {
			return err
		}
		fmt.Printf("acked %s\n", args[0])
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

// firstLine returns the first line of s, marked with an ellipsis if truncated.
func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i] + " …"
	}
	return s
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
	learningsPeekCmd.Flags().Bool("json", false, "emit pending items as JSON")
	learningsPeekCmd.Flags().String("channel", "", "filter to one channel (e.g. learning)")
	learningsCmd.AddCommand(learningsStatusCmd, learningsPeekCmd, learningsAckCmd, learningsEnqueueCmd)
}
