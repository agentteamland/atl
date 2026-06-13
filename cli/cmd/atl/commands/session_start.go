package commands

import (
	"fmt"
	"time"

	"github.com/agentteamland/atl/cli/internal/doctor"
	"github.com/spf13/cobra"
)

var sessionStartCmd = &cobra.Command{
	Use:   "session-start",
	Short: "Session-start maintenance (run by the SessionStart hook)",
	Long: "Run by the SessionStart hook. Drains the previous session's transcripts\n" +
		"into the queue and runs the doctor self-check. Whatever it prints reaches\n" +
		"Claude's context (SessionStart is one of the events that delivers stdout),\n" +
		"so it stays quiet unless there's something worth surfacing.\n\n" +
		"Never fails: a hook must not block the session. (Network update lands here\n" +
		"once update is real.)",
	RunE: func(cmd *cobra.Command, args []string) error {
		st, project, err := openQueue()
		if err != nil {
			return nil // non-blocking: never fail a hook
		}
		defer st.Close()

		// Drain the previous session's transcripts (no throttle at session start).
		if _, _, enqueued, derr := drainProjectTranscripts(st, project); derr == nil && enqueued > 0 {
			fmt.Printf("atl: captured %d new learning(s) from the previous session\n", enqueued)
		}

		// Doctor self-check — surface only non-OK results.
		for _, r := range doctor.Run(doctor.QueueChecks(st, project, time.Now())) {
			if r.Status != doctor.OK {
				fmt.Printf("atl doctor: %s — %s\n", r.Status, r.Detail)
			}
		}
		return nil
	},
}
