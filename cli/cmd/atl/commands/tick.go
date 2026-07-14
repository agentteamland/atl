package commands

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"time"

	"github.com/agentteamland/atl/cli/internal/doctor"
	"github.com/agentteamland/atl/cli/internal/drain"
	"github.com/agentteamland/atl/cli/internal/generation"
	"github.com/agentteamland/atl/cli/internal/queue"
	"github.com/agentteamland/atl/cli/internal/throttle"
	"github.com/agentteamland/atl/cli/internal/transcript"
	"github.com/spf13/cobra"
)

var tickCmd = &cobra.Command{
	Use:   "tick",
	Short: "Run the in-session maintenance tick",
	Long: "Run the in-session maintenance tick — the work the three-speed cadence\n" +
		"fires every few minutes via the prompt-piggyback throttle: a cheap\n" +
		"every-call fan-out (guarded by the global generation counter, so it's a\n" +
		"no-op when nothing changed) plus a throttled drain + doctor self-check.\n\n" +
		"By default it discovers this project's Claude Code transcripts (modified\n" +
		"since the last tick), extracts the assistant text, and transfers any capture\n" +
		"markers into the durable queue — exactly once. --throttle skips the heavier\n" +
		"drain+doctor pass if the last tick was within the given duration (how the\n" +
		"per-prompt hook stays cheap); the fan-out still runs (it's already ~free).\n" +
		"--file drains a single file instead (manual/debug).",
	RunE: func(cmd *cobra.Command, args []string) error {
		st, project, err := openQueue()
		if err != nil {
			return err
		}
		defer st.Close()

		// Manual override: drain a single file (test/debug), no throttle/cursor.
		if file, _ := cmd.Flags().GetString("file"); file != "" {
			b, err := os.ReadFile(file)
			if err != nil {
				return fmt.Errorf("read --file: %w", err)
			}
			r, err := drain.Drain(string(b), project, st)
			if err != nil {
				return err
			}
			fmt.Printf("tick: drained %s — %d marker(s), %d new, %d already queued\n",
				file, r.Found, r.Enqueued, r.Found-r.Enqueued)
			return nil
		}

		// Every-call fan-out, generation-guarded: a no-op (one small file read)
		// when the global layer hasn't changed since this project last fanned out,
		// so it's cheap enough to ride every prompt. A fan-out error is surfaced
		// (not swallowed) so a persistently failing pass is observable.
		if changed, gen, gerr := generation.Changed(project); gerr == nil && changed {
			if n, ferr := fanOut(project); ferr == nil {
				if n > 0 {
					fmt.Printf("tick: fanned out %d file(s) from the global layer\n", n)
				}
				_ = generation.MarkSeen(project, gen)
			} else {
				fmt.Printf("tick: fan-out skipped this pass (%v)\n", ferr)
			}
		}

		// Auto-drain signal (unthrottled, every turn): if the queue holds pending
		// learnings, tell the agent to drain them now in a background subagent. This
		// is the deterministic per-turn trigger the learning-capture rule acts on —
		// the CLI half of automatic integration (the drain itself is the agent's LLM
		// work). Cheap (one count read); placed before the throttle gate so it fires
		// on every prompt the queue is non-empty, not just when the heavier pass runs.
		if counts, cerr := st.Counts(project); cerr == nil {
			if msg := autoDrainNotice(counts[queue.ChannelLearning]); msg != "" {
				fmt.Println(msg)
			}
		}

		// Throttle gate (auto mode): skip the heavier drain+doctor pass if we ran
		// it too recently. The stamp is per-project so concurrent sessions in
		// different projects don't starve each other's drain/doctor/promote pass.
		throttleDur, _ := cmd.Flags().GetDuration("throttle")
		var stamp string
		if throttleDur > 0 {
			if stamp, err = throttle.StampPath("last-tick-" + projectStamp(project)); err != nil {
				return err
			}
			if !throttle.Gate(stamp, throttleDur) {
				return nil
			}
		}

		scanned, found, enqueued, skipped, err := drainProjectTranscripts(st, project)
		if err != nil {
			return err
		}
		if skipped > 0 {
			fmt.Printf("tick: skipped %d unreadable transcript(s) this pass\n", skipped)
		}

		// Doctor self-check (queue health + asset integrity + hook binding), same
		// as session-start.
		for _, r := range doctor.Run(append(doctor.QueueChecks(st, project, time.Now()), integrityCheck(project), hooksCheck())) {
			if r.Status != doctor.OK || r.Healed {
				fmt.Printf("tick doctor: %s — %s\n", r.Status, r.Detail)
			}
		}

		// Lift this project's accumulated gains to the global layer (ring 1→2).
		// Auto + visible (decision doc 5.1): additive, conflict-archived, and
		// pinnable, so it's risk-free enough to ride the tick instead of waiting
		// for a manual `atl promote`. Quiet when there's nothing to lift; a promote
		// error is surfaced rather than swallowed.
		if pr, perr := promoteGains(project, ""); perr != nil {
			fmt.Printf("tick: promote skipped this pass (%v)\n", perr)
		} else if pr.lifted > 0 {
			fmt.Printf("tick: %s\n", pr.String())
		}

		if throttleDur > 0 {
			_ = throttle.Touch(stamp)
		}
		if scanned == 0 {
			fmt.Println("tick: no new transcripts to drain")
			return nil
		}
		fmt.Printf("tick: scanned %d transcript(s) — %d marker(s), %d new, %d already queued\n",
			scanned, found, enqueued, found-enqueued)
		return nil
	},
}

// drainProjectTranscripts discovers this project's transcripts modified since
// the cursor, drains each into the queue, and advances the cursor. Shared by
// `atl tick` and `atl session-start`.
//
// Fail-soft per file: one unreadable/undrainable transcript is skipped (counted
// as skipped), never aborting the batch — otherwise a single poison file wedges
// the project's entire capture pipeline forever (the cursor never advances, so
// every later transcript is re-blocked). Queue dedup makes any re-scan a no-op,
// so advancing past a skipped file is safe.
func drainProjectTranscripts(st *queue.Store, project string) (scanned, found, enqueued, skipped int, err error) {
	dir, err := transcript.ProjectDir(project)
	if err != nil {
		return 0, 0, 0, 0, err
	}
	since, err := st.Cursor(project)
	if err != nil {
		return 0, 0, 0, 0, err
	}
	files, err := transcript.Find(dir, since)
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("find transcripts: %w", err)
	}
	var newest time.Time
	for _, f := range files {
		if f.ModTime.After(newest) {
			newest = f.ModTime
		}
		text, e := transcript.ExtractText(f.Path)
		if e != nil {
			skipped++
			continue
		}
		r, e := drain.Drain(text, project, st)
		if e != nil {
			skipped++
			continue
		}
		found += r.Found
		enqueued += r.Enqueued
	}
	scanned = len(files)
	if scanned > 0 {
		// Clamp a future modtime (a wedge otherwise: the cursor would sit ahead of
		// wall-clock forever) and back off a small slack so same-second modtime ties
		// (coarse-FS concurrent appends) re-scan rather than skip — free via dedup.
		now := time.Now()
		if newest.After(now) {
			newest = now
		}
		if e := st.SetCursor(project, newest.Add(-time.Second)); e != nil {
			return scanned, found, enqueued, skipped, fmt.Errorf("advance cursor: %w", e)
		}
	}
	// Record that the maintenance pass ran, for doctor's tick-freshness check.
	_ = st.SetLastTick(project, time.Now())
	return scanned, found, enqueued, skipped, nil
}

// projectStamp is a short, filesystem-safe token derived from a project path,
// used to give each project its own throttle stamp under ~/.atl/cache.
func projectStamp(project string) string {
	sum := sha256.Sum256([]byte(project))
	return hex.EncodeToString(sum[:])[:16]
}

func init() {
	tickCmd.Flags().String("file", "", "drain a single file instead of discovering transcripts (manual/debug)")
	tickCmd.Flags().Duration("throttle", 0, "skip the drain+doctor pass if the last tick was within this duration (e.g. 10m)")
}
