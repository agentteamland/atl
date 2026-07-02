package commands

import (
	"fmt"
	"time"

	"github.com/agentteamland/atl/cli/internal/doctor"
	"github.com/agentteamland/atl/cli/internal/gc"
	"github.com/agentteamland/atl/cli/internal/queue"
	"github.com/spf13/cobra"
)

var sessionStartCmd = &cobra.Command{
	Use:   "session-start",
	Short: "Session-start maintenance (run by the SessionStart hook)",
	Long: "Run by the SessionStart hook. Drains the previous session's transcripts\n" +
		"into the queue, runs the doctor self-check, and signals any pending\n" +
		"learnings so Claude can fold them into the knowledge base via /drain.\n" +
		"Whatever it prints reaches Claude's context (SessionStart delivers stdout),\n" +
		"so it stays quiet unless there's something worth surfacing.\n\n" +
		"Never fails: a hook must not block the session.",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Refresh the platform core (rules + skills) into the global layer — it
		// ships in the binary, so this keeps ~/.claude in lockstep with the binary
		// version. Non-blocking; a hook must never fail.
		if n, _ := reflectCore(); n > 0 {
			fmt.Printf("atl: refreshed %d core file(s)\n", n)
		}

		st, project, err := openQueue()
		if err != nil {
			return nil // non-blocking: never fail a hook
		}
		defer st.Close()

		// Drain the previous session's transcripts (no throttle at session start).
		if _, _, enqueued, derr := drainProjectTranscripts(st, project); derr == nil && enqueued > 0 {
			fmt.Printf("atl: captured %d new learning(s) from the previous session\n", enqueued)
		}

		// Doctor self-check + asset integrity restore — surface non-OK / healed.
		checks := append(doctor.QueueChecks(st, project, time.Now()), integrityCheck(project))
		for _, r := range doctor.Run(checks) {
			if r.Status != doctor.OK || r.Healed {
				fmt.Printf("atl doctor: %s — %s\n", r.Status, r.Detail)
			}
		}

		// Reclamation awareness — surface only high-signal orphans (gains/edits
		// beside an installed unit), not wholly-unowned dirs (usually the user's own
		// non-ATL Claude Code assets — noise). Awareness only; `atl gc` is the action.
		if orphans, oerr := gc.Scan(project, time.Now()); oerr == nil {
			n := 0
			for _, o := range orphans {
				if o.Owned {
					n++
				}
			}
			if n > 0 {
				fmt.Printf("atl: %d orphaned file(s) beside installed units — run `atl gc` to review (reversible)\n", n)
			}
		}

		// Signal pending learnings so Claude folds them in via /drain. The skill is
		// LLM work the CLI can't run itself (the CLI/Skill boundary) — surfacing the
		// count here is how it gets triggered without the user remembering to.
		if counts, cerr := st.Counts(project); cerr == nil {
			if n := counts[queue.ChannelLearning]; n > 0 {
				fmt.Printf("atl: %d learning(s) pending — run /drain to fold them into the knowledge base\n", n)
			}
			// profile-team's channel: only fires when profile-team is installed and a
			// session dropped profile-fact markers. /profile-drain is a team skill; core
			// /drain stays learning-only (its documented boundary).
			if n := counts[queue.ChannelProfileFact]; n > 0 {
				fmt.Printf("atl: %d profile-fact(s) pending — run /profile-drain to fold them into the profiles\n", n)
			}
		}

		// Docs-correctness signal — fires only in a repo that has a docs site
		// (monorepo-internal): a deterministic drift count plus a /docs-audit
		// "sweep due" signal. Silent everywhere else.
		docsSessionSignal()

		// Skill/asset content-quality signal — monorepo-internal, same as docs.
		skillsSessionSignal()

		// Rules-distill "distill due" signal — monorepo-internal, same shape.
		rulesSessionSignal()

		return nil
	},
}
