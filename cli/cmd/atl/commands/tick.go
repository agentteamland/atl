package commands

import (
	"fmt"
	"os"
	"time"

	"github.com/agentteamland/atl/cli/internal/drain"
	"github.com/agentteamland/atl/cli/internal/transcript"
	"github.com/spf13/cobra"
)

var tickCmd = &cobra.Command{
	Use:   "tick",
	Short: "Run the in-session maintenance tick",
	Long: "Run the in-session maintenance tick — the work the three-speed cadence\n" +
		"fires every few minutes via the prompt-piggyback throttle. The v2 target is\n" +
		"drain + doctor + fan-out; this wires the drain.\n\n" +
		"By default it discovers this project's Claude Code transcripts (those\n" +
		"modified since the last tick), extracts the assistant text, and transfers\n" +
		"any capture markers into the durable queue — exactly once, so re-running is\n" +
		"a safe no-op. --file drains a single file instead (manual/debug).",
	RunE: func(cmd *cobra.Command, args []string) error {
		st, project, err := openQueue()
		if err != nil {
			return err
		}
		defer st.Close()

		// Manual override: drain a single file (test/debug), no cursor.
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

		// Default: discover + drain this project's transcripts since the cursor.
		dir, err := transcript.ProjectDir(project)
		if err != nil {
			return err
		}
		since, err := st.Cursor(project)
		if err != nil {
			return err
		}
		files, err := transcript.Find(dir, since)
		if err != nil {
			return fmt.Errorf("find transcripts: %w", err)
		}
		if len(files) == 0 {
			fmt.Println("tick: no new transcripts to drain")
			return nil
		}

		var found, enqueued int
		var newest time.Time
		for _, f := range files {
			text, err := transcript.ExtractText(f.Path)
			if err != nil {
				return fmt.Errorf("read %s: %w", f.Path, err)
			}
			r, err := drain.Drain(text, project, st)
			if err != nil {
				return err
			}
			found += r.Found
			enqueued += r.Enqueued
			if f.ModTime.After(newest) {
				newest = f.ModTime
			}
		}
		if err := st.SetCursor(project, newest); err != nil {
			return fmt.Errorf("advance cursor: %w", err)
		}
		fmt.Printf("tick: scanned %d transcript(s) — %d marker(s), %d new, %d already queued\n",
			len(files), found, enqueued, found-enqueued)
		return nil
	},
}

func init() {
	tickCmd.Flags().String("file", "", "drain a single file instead of discovering transcripts (manual/debug)")
}
