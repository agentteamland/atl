package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
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
		for _, dc := range declaredChannels(project) {
			ch := queue.Channel(dc.Name)
			if n, ok := counts[ch]; ok {
				fmt.Printf("  %-14s %d\n", string(ch), n)
				delete(counts, ch)
			}
		}
		// Anything left is an orphan: items queued on a channel no longer active,
		// e.g. because the team that owned it was uninstalled while items were
		// pending. Printed anyway — a count that vanished from the report would be
		// items nobody can see.
		for ch, n := range counts {
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
		items, err := st.Pending(project, queue.Channel(channel))
		if err != nil {
			return err
		}
		// Reject an undeclared channel rather than silently returning an empty list —
		// a typo like `--channel learnings` would otherwise look like "nothing pending".
		// The error enumerates what is actually active here, which depends on which
		// teams are installed.
		//
		// Checked only when the filter matched nothing, so this stays a gate on typos
		// and never on data: items already in the queue on a channel that is no longer
		// active — the team was uninstalled, or its declaration has not been backfilled
		// yet — are real, and `learnings status` reports them for exactly that reason. A
		// read must not hide what the count admits exists. A typo matches nothing, so it
		// still fails loudly here.
		if chans := declaredChannels(project); channel != "" && len(items) == 0 && !isDeclaredChannel(chans, channel) {
			return fmt.Errorf("unknown --channel %q (active: %s)", channel, strings.Join(channelNames(chans), ", "))
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

// learningsEnqueueCmd is a hidden helper: hooks, the /drain skill's mining step,
// and tests use it to transfer a captured marker into the queue exactly once.
// The dedup lives in the store, so calling it twice with the same marker is a
// safe no-op.
//
// It is one of exactly two write seams into the queue (the other is
// drain.Drain), so it validates the channel: an undeclared name would otherwise
// leave pending items in the project's bucket that no drain will ever claim —
// the marker LOOKS captured while nothing can ever process it.
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

		if chans := declaredChannels(project); !isDeclaredChannel(chans, args[0]) {
			return fmt.Errorf("unknown channel %q (active: %s)", args[0], strings.Join(channelNames(chans), ", "))
		}
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

// learningsTranscriptCmd prints the user+assistant conversation flow so a drain
// skill's mining step can spot corrections, reverts, and durable facts the agent
// never marked — the missing input side of the marker-only capture. Anything
// mined from it is enqueued like a marker, so dedup lives in the queue (content
// hash) and re-reading is always safe.
//
// It has two modes, and the difference is the cursor:
//
//   - Bare, it is a plain read of the most recent prose — no state, safe to run
//     for a look, and the mode the profile curator's corroboration lookup wants.
//   - With --channel, it is that channel's SWEEP: it resumes where that channel's
//     last sweep stopped and advances the cursor. Only a sweep may advance, so an
//     ad-hoc read can never consume a drain's unmined window.
var learningsTranscriptCmd = &cobra.Command{
	Use:   "transcript",
	Short: "Print conversation flow (for a drain's mining step)",
	Long: "Emit the user+assistant conversation flow for the current project so a drain\n" +
		"skill can mine corrections, reverts, and durable facts the agent never marked.\n" +
		"Tool calls/results are dropped — prose only; --json emits role/text records.\n\n" +
		"Bare, it reads the most recent 256 KB of prose from the most recent --limit\n" +
		"transcripts (default 2) and records nothing.\n\n" +
		"--channel <name> makes it that channel's sweep instead: it resumes from where\n" +
		"that channel last mined, emits the next 256 KB of prose FORWARD, advances the\n" +
		"channel's cursor, and reports what is still pending. Successive sweeps then\n" +
		"cover a session completely instead of re-reading its tail; --limit does not\n" +
		"apply, since a transcript with unmined turns is never skipped for being old.",
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

		channel, _ := cmd.Flags().GetString("channel")
		var turns []transcript.Turn
		var overLong int
		if channel == "" {
			turns, overLong = readFlowTail(cmd, files)
		} else {
			if chans := declaredChannels(project); !isDeclaredChannel(chans, channel) {
				return fmt.Errorf("unknown channel %q (active: %s)", channel, strings.Join(channelNames(chans), ", "))
			}
			turns, overLong, err = sweepFlow(cmd, project, channel, files)
			if err != nil {
				return err
			}
		}

		if overLong > 0 {
			fmt.Fprintf(cmd.ErrOrStderr(), "atl: skipped %d over-long transcript record(s) — their turns are missing from this flow\n", overLong)
		}
		if jsonOut, _ := cmd.Flags().GetBool("json"); jsonOut {
			b, err := json.MarshalIndent(turns, "", "  ")
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), string(b))
			return nil
		}
		if len(turns) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "no recent conversation flow")
			return nil
		}
		for _, t := range turns {
			fmt.Fprintf(cmd.OutOrStdout(), "[%s] %s\n", t.Role, t.Text)
		}
		return nil
	},
}

// readFlowTail is the cursorless read: the most recent maxFlowBytes of prose
// from the most recent --limit transcripts. It records nothing, so running it
// cannot move any channel's sweep.
func readFlowTail(cmd *cobra.Command, files []transcript.File) ([]transcript.Turn, int) {
	limit, _ := cmd.Flags().GetInt("limit")
	if limit > 0 && len(files) > limit {
		files = files[len(files)-limit:] // Find returns oldest-first → keep the most recent N
	}
	var turns []transcript.Turn
	var overLong int
	for _, fl := range files {
		t, dropped, err := transcript.ExtractFlow(fl.Path)
		overLong += dropped
		if err != nil {
			continue // a single unreadable transcript shouldn't fail the mine
		}
		turns = append(turns, t...)
	}
	turns, truncated := tailByBytes(turns, maxFlowBytes)
	// The notes go to stderr: --json must stay a parseable array (the whole point
	// of the flag), and an agent's shell shows both streams anyway, so the skill
	// still reads them and can report an honest partial sweep.
	if truncated {
		fmt.Fprintf(cmd.ErrOrStderr(), "atl: flow truncated to the most recent %d KB of prose — older turns were not emitted; report this mine as a partial sweep\n", maxFlowBytes/1024)
	}
	return turns, overLong
}

// sweepFlow is the cursored read: it walks the project's transcripts oldest-first,
// resumes each from where this channel last mined, and spends one maxFlowBytes
// budget going FORWARD. Whatever does not fit stays pending and is reported —
// the difference between a bounded sweep and a silent gap.
//
// The cursor advances only over turns actually emitted, and only for a file that
// read cleanly, so a read error re-reads rather than skips. On the first sweep a
// channel has no baseline: every existing transcript is stamped at its current
// end and the run falls back to the tail read, so adopting the cursor does not
// replay every session the project has ever had.
func sweepFlow(cmd *cobra.Command, project, channel string, files []transcript.File) ([]transcript.Turn, int, error) {
	cur, err := transcript.LoadCursor(project)
	if err != nil {
		return nil, 0, err
	}
	keep := make(map[string]bool, len(files))
	for _, fl := range files {
		keep[filepath.Base(fl.Path)] = true
	}

	if !cur.Known(channel) {
		for _, fl := range files {
			cur.SetOffset(channel, filepath.Base(fl.Path), fl.Size)
		}
		if err := cur.Save(project); err != nil {
			return nil, 0, err
		}
		fmt.Fprintf(cmd.ErrOrStderr(), "atl: first sweep for channel %q — read the recent tail and baselined %d transcript(s); the next sweep resumes forward from here\n", channel, len(files))
		turns, overLong := readFlowTail(cmd, files)
		return turns, overLong, nil
	}

	var turns []transcript.Turn
	var overLong int
	budget := maxFlowBytes
	for _, fl := range files {
		base := filepath.Base(fl.Path)
		off, _ := cur.Offset(channel, base) // absent in a known channel = new transcript, mine from 0
		if off > fl.Size {
			off = 0 // truncated or replaced under the same name — re-read it whole
			cur.SetOffset(channel, base, 0)
		}
		if off >= fl.Size || budget <= 0 {
			continue
		}
		flow, err := transcript.ExtractFlowFrom(fl.Path, off, budget)
		overLong += flow.Skipped
		if err != nil {
			continue // leave the cursor where it was: a failed read is retried, never skipped
		}
		for _, t := range flow.Turns {
			budget -= len(t.Text)
		}
		turns = append(turns, flow.Turns...)
		cur.SetOffset(channel, base, flow.Next)
		if flow.Pending {
			budget = 0
		}
	}

	cur.Prune(channel, keep)
	if err := cur.Save(project); err != nil {
		return nil, 0, err
	}

	var pending int64
	for _, fl := range files {
		if off, _ := cur.Offset(channel, filepath.Base(fl.Path)); fl.Size > off {
			pending += fl.Size - off
		}
	}
	if pending > 0 {
		fmt.Fprintf(cmd.ErrOrStderr(), "atl: %s of transcript still unmined for channel %q — sweep again to continue (nothing is lost; the cursor holds the place)\n", humanBytes(pending), channel)
	}
	return turns, overLong, nil
}

// maxFlowBytes caps the prose `transcript` emits. Its only other bound is a file
// count (--limit), and a file is not a size: five real sessions measured here
// carried 131-633 KB of prose each, so the default two-file read can hand the
// mining step ~900 KB — more than the subagent's whole context. Dilution is the
// second cost: the step looks for what went UNMARKED, and a bigger haystack with
// the same needle count makes it least reliable exactly when the session was long
// enough to have forgotten something. 256 KB is ~64k tokens at ~4 bytes/token —
// it kept 164-552 turns of those same sessions, leaving room for the skill's own
// instructions. Fixed until a v2 config layer exists, like the watchdog thresholds.
const maxFlowBytes = 256 * 1024

// tailByBytes keeps the most recent turns that fit in maxBytes and reports
// whether anything older was cut. Recent-first because an unmarked correction is
// most likely in the stretch just gone by. The newest turn is always kept: one
// oversized turn must not blank the mining input entirely.
func tailByBytes(turns []transcript.Turn, maxBytes int) ([]transcript.Turn, bool) {
	total := 0
	for i := len(turns) - 1; i >= 0; i-- {
		total += len(turns[i].Text)
		if total > maxBytes && i < len(turns)-1 {
			return turns[i+1:], true
		}
	}
	return turns, false
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

// learningsRecoverCmd moves pending items out of buckets whose project directory
// no longer exists and into this project's bucket, where a drain can reach them.
//
// It exists because the doctor's stranded-bucket warning would otherwise have no
// remedy, and a warning nobody can clear becomes wallpaper — this repo has
// measured that failure on its own signals. Explicit rather than automatic: the
// items land in whatever project you are standing in, which is a judgement about
// where that knowledge belongs, not something to guess.
var learningsRecoverCmd = &cobra.Command{
	Use:   "recover",
	Short: "Move pending items from deleted projects into this one, so a drain can reach them",
	Long: "Buckets keyed to a directory that no longer exists — a deleted worktree, a throwaway\n" +
		"checkout — hold items no project-scoped surface can name. This moves them\n" +
		"into the current project's bucket. Payload, id and original capture time are preserved,\n" +
		"so the drain sees a rescue rather than a re-capture. Dry-run unless --apply.",
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		apply, _ := cmd.Flags().GetBool("apply")
		st, project, err := openQueue()
		if err != nil {
			return err
		}
		defer st.Close()

		projects, err := st.Projects()
		if err != nil {
			return err
		}
		var stranded []string
		items := 0
		for key, counts := range projects {
			if key == project {
				continue
			}
			if fi, serr := os.Stat(key); serr == nil && fi.IsDir() {
				continue
			}
			stranded = append(stranded, key)
			for _, n := range counts {
				items += n
			}
		}
		sort.Strings(stranded)

		if len(stranded) == 0 {
			fmt.Println("atl learnings: nothing stranded — every pending item is in a live project")
			return nil
		}
		for _, k := range stranded {
			parts := make([]string, 0, len(projects[k]))
			for ch, n := range projects[k] {
				parts = append(parts, fmt.Sprintf("%s:%d", ch, n))
			}
			sort.Strings(parts)
			fmt.Printf("  [%s] %s\n", strings.Join(parts, " "), k)
		}
		if !apply {
			fmt.Printf("atl learnings: %d item(s) in %d deleted project(s) — dry run; re-run with --apply to move them into %s\n", items, len(stranded), project)
			return nil
		}
		moved, err := st.Recover(project, stranded)
		if err != nil {
			return err
		}
		fmt.Printf("atl learnings: recovered %d item(s) into %s — run a drain to process them\n", moved, project)
		return nil
	},
}

// openQueue opens the default queue and resolves the current project key.
func openQueue() (*queue.Store, string, error) {
	dbPath, err := queue.DefaultPath()
	if err != nil {
		return nil, "", err
	}
	st, err := queue.Open(dbPath)
	if err != nil {
		return nil, "", err
	}
	project, err := projectKey()
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
	learningsTranscriptCmd.Flags().Int("limit", 2, "read the most recent N transcripts (cursorless read only)")
	learningsTranscriptCmd.Flags().String("channel", "", "sweep forward for this capture channel, advancing its cursor")
	learningsRecoverCmd.Flags().Bool("apply", false, "actually move the items (default is a dry run)")
	learningsCmd.AddCommand(learningsStatusCmd, learningsPeekCmd, learningsAckCmd, learningsEnqueueCmd, learningsTranscriptCmd, learningsRecoverCmd)
}
