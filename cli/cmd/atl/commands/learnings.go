package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/agentteamland/atl/cli/internal/queue"
	"github.com/agentteamland/atl/cli/internal/transcript"
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
	Args: cobra.NoArgs,
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
		if jsonOut, _ := cmd.Flags().GetBool("json"); jsonOut {
			out, err := statusJSON(counts)
			if err != nil {
				return err
			}
			fmt.Println(out)
			return nil
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
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		st, project, err := openQueue()
		if err != nil {
			return err
		}
		defer st.Close()

		channel, _ := cmd.Flags().GetString("channel")
		// Reject an unknown channel rather than silently returning an empty list —
		// a typo like `--channel learnings` would otherwise look like "nothing pending".
		if channel != "" && !knownChannel(queue.Channel(channel)) {
			return fmt.Errorf("unknown --channel %q (known: %s, %s)", channel, queue.ChannelLearning, queue.ChannelProfileFact)
		}
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
			fmt.Printf("%-12s  %-12s  %s\n", shortID(it.ID), it.Channel, firstLine(it.Payload))
		}
		return nil
	},
}

// learningsAckCmd deletes a processed item — processed-then-deleted, so it can
// never be re-reported. The /drain skill calls this after integrating an item.
// The id may be a full id or an unambiguous prefix (peek shows a short prefix),
// resolved git-style against the pending set so a wrong/ambiguous id fails loudly
// instead of silently no-op'ing (bbolt Delete on a missing key is a no-op).
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
		items, err := st.Pending(project, "")
		if err != nil {
			return err
		}
		id, err := resolveAckID(items, args[0])
		if err != nil {
			return err
		}
		if err := st.Delete(project, id); err != nil {
			return err
		}
		fmt.Printf("acked %s\n", id)
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

// learningsTranscriptCmd prints the recent user+assistant conversation flow so
// the /drain skill's correction-mining step can spot user corrections, reverts,
// and repeated mistakes the agent never marked — the missing input side of the
// marker-only capture. Anything mined from this is enqueued like a marker, so
// dedup lives in the queue (content hash); this is a plain read with no cursor
// to advance.
var learningsTranscriptCmd = &cobra.Command{
	Use:   "transcript",
	Short: "Print recent conversation flow (for /drain correction-mining)",
	Long: "Emit the recent user+assistant conversation flow for the current project so\n" +
		"the /drain skill can mine user corrections, reverts, and repeated mistakes the\n" +
		"agent never marked. Tool calls/results are dropped — prose only. --limit N reads\n" +
		"the most recent N transcripts (default 2); --json emits role/text records.",
	RunE: func(cmd *cobra.Command, args []string) error {
		project, err := os.Getwd()
		if err != nil {
			return err
		}
		dir, err := transcript.ProjectDir(project)
		if err != nil {
			return err
		}
		files, err := transcript.Find(dir, time.Time{})
		if err != nil {
			return err
		}
		limit, _ := cmd.Flags().GetInt("limit")
		if limit > 0 && len(files) > limit {
			files = files[len(files)-limit:] // Find returns oldest-first → keep the most recent N
		}
		var turns []transcript.Turn
		for _, fl := range files {
			t, err := transcript.ExtractFlow(fl.Path)
			if err != nil {
				continue // a single unreadable transcript shouldn't fail the mine
			}
			turns = append(turns, t...)
		}
		if jsonOut, _ := cmd.Flags().GetBool("json"); jsonOut {
			b, err := json.MarshalIndent(turns, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(b))
			return nil
		}
		if len(turns) == 0 {
			fmt.Println("no recent conversation flow")
			return nil
		}
		for _, t := range turns {
			fmt.Printf("[%s] %s\n", t.Role, t.Text)
		}
		return nil
	},
}

// statusJSON marshals the per-channel pending counts to a stable JSON object
// (channel→count). An empty or nil queue marshals to "{}" rather than JSON
// null, so a caller always gets an object. queue.Channel is a string type, so
// encoding/json emits the keys in sorted order — deterministic output tooling
// can rely on.
func statusJSON(counts map[queue.Channel]int) (string, error) {
	if len(counts) == 0 {
		return "{}", nil
	}
	b, err := json.Marshal(counts)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// knownChannel reports whether ch is a queue channel the platform recognizes.
func knownChannel(ch queue.Channel) bool {
	return ch == queue.ChannelLearning || ch == queue.ChannelProfileFact
}

// shortID is the 12-char id prefix shown by peek (and echoed in ack's ambiguity
// error). Guarded so it never panics on an unexpectedly short id.
func shortID(id string) string {
	if len(id) <= 12 {
		return id
	}
	return id[:12]
}

// resolveAckID resolves an id or an unambiguous id-prefix against the pending
// items, git-short-SHA style: it returns the single full ID that idArg is a
// prefix of. peek displays a 12-char prefix, so copy-pasting that into ack must
// resolve to the right item; and a non-matching or ambiguous id must fail loudly
// rather than let ack print "acked" on a Delete that removed nothing.
func resolveAckID(items []queue.Item, idArg string) (string, error) {
	idArg = strings.TrimSpace(idArg)
	if idArg == "" {
		return "", fmt.Errorf("ack: empty id")
	}
	var matches []string
	for _, it := range items {
		if strings.HasPrefix(it.ID, idArg) {
			matches = append(matches, it.ID)
		}
	}
	switch len(matches) {
	case 1:
		return matches[0], nil
	case 0:
		return "", fmt.Errorf("ack: no queued item matches id %q (run `atl learnings peek` to list ids)", idArg)
	default:
		shorts := make([]string, len(matches))
		for i, id := range matches {
			shorts[i] = shortID(id)
		}
		return "", fmt.Errorf("ack: ambiguous id %q — matches %d items: %s", idArg, len(matches), strings.Join(shorts, ", "))
	}
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
	learningsStatusCmd.Flags().Bool("json", false, "emit pending counts as JSON")
	learningsPeekCmd.Flags().Bool("json", false, "emit pending items as JSON")
	learningsPeekCmd.Flags().String("channel", "", "filter to one channel (e.g. learning)")
	learningsTranscriptCmd.Flags().Bool("json", false, "emit turns as JSON")
	learningsTranscriptCmd.Flags().Int("limit", 2, "read the most recent N transcripts")
	learningsCmd.AddCommand(learningsStatusCmd, learningsPeekCmd, learningsAckCmd, learningsEnqueueCmd, learningsTranscriptCmd)
}
